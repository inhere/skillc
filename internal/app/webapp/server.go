package webapp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/domain/skill"
)

// FileEntry represents a row in the sidebar file list.
type FileEntry struct {
	RelPath  string // relative path from skill root (e.g. "SKILL.md", "examples/foo.md")
	Indent   int    // depth level for display indentation
	IsDir    bool   // true when this row is a directory label, not a clickable file
	DirLabel string // set when IsDir=true
}

// fileContentResp is the JSON payload returned by the /file endpoint.
type fileContentResp struct {
	Content    string `json:"content"`
	IsMarkdown bool   `json:"isMarkdown"`
	Error      string `json:"error,omitempty"`
}

// Server is the skill web viewer. It serves a local HTTP server that lets
// the user browse a skill's files with rendered Markdown.
type Server struct {
	// Out is where startup messages are written. Defaults to os.Stdout.
	Out io.Writer
}

// NewServer creates a Server that writes startup messages to stdout.
func NewServer() *Server {
	return &Server{Out: os.Stdout}
}

// Serve starts the HTTP server on the given port and blocks until it exits.
func (s *Server) Serve(item skill.Skill, port int) error {
	skillDir := item.Path
	if skillDir == "" {
		return fmt.Errorf("skill %q has no local path; cannot start web viewer", item.ID)
	}

	entries, err := ListFileEntries(skillDir)
	if err != nil {
		return fmt.Errorf("failed to list skill files: %w", err)
	}

	defaultFile := "SKILL.md"
	if !EntryContains(entries, defaultFile) {
		if len(entries) > 0 {
			defaultFile = entries[0].RelPath
		} else {
			defaultFile = ""
		}
	}

	funcMap := template.FuncMap{
		"skillFileIcon": FileIcon,
		"indent":        func(depth int) int { return 14 + depth*12 },
		"indentFile":    func(depth int) int { return 14 + depth*12 },
		"fileName": func(relPath string) string {
			return filepath.Base(filepath.FromSlash(relPath))
		},
	}
	tmpl, err := template.New("skill-web").Funcs(funcMap).Parse(htmlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse web template: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.Execute(w, map[string]interface{}{
			"SkillID":     item.ID,
			"SkillName":   item.Name,
			"Description": item.Description,
			"Entries":     entries,
			"DefaultFile": defaultFile,
		})
	})

	// /file serves raw file content; paths are confined to skillDir.
	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		resp := fileContentResp{}

		relPath := r.URL.Query().Get("name")
		abs, err := checkQueryFilePath(skillDir, relPath)
		if err != nil {
			ccolor.Errorf("invalid path, %v", err.Error())
			resp.Error = "invalid path"
			jsonResponse(w, http.StatusBadRequest, resp)
			return
		}

    // read file content
		content, err := os.ReadFile(abs)
		if err != nil {
			resp.Error = err.Error()
			w.WriteHeader(http.StatusNotFound)
		} else {
			resp.Content = string(content)
			resp.IsMarkdown = strings.HasSuffix(strings.ToLower(relPath), ".md")
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	addr := fmt.Sprintf(":%d", port)
	ccolor.Fprintf(s.Out, "Skill web viewer started: <info>http://localhost%s</>\n", addr)
	ccolor.Fprintf(s.Out, "Skill: <info>%s (%s)</>\n", item.Name, item.ID)
	ccolor.Fprintln(s.Out, "Press <yellow>Ctrl+C</> to stop")
	return http.ListenAndServe(addr, mux)
}

func checkQueryFilePath(skillDir, relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("name query parameter is required")
	}

	rootAbs, err := filepath.Abs(skillDir)
	if err != nil {
		return "", err
	}
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}

	candidate := filepath.Clean(filepath.Join(rootEval, filepath.FromSlash(relPath)))
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	abs, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return "", err
	}

	relToRoot, err := filepath.Rel(rootEval, abs)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", err
	}
	return abs, nil
}

func jsonResponse(w http.ResponseWriter, status int, resp any) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListFileEntries recursively collects all non-hidden files under root,
// returning a flat list with directory label rows inserted for grouping.
// SKILL.md is always promoted to the top of the list.
func ListFileEntries(root string) ([]FileEntry, error) {
	var result []FileEntry
	if err := walkDir(root, root, 0, &result); err != nil {
		return nil, err
	}

	// Promote SKILL.md to the very top
	for i, e := range result {
		if !e.IsDir && e.RelPath == "SKILL.md" && i > 0 {
			entry := result[i]
			result = append([]FileEntry{entry}, append(result[:i], result[i+1:]...)...)
			break
		}
	}
	return result, nil
}

func walkDir(root, dir string, depth int, result *[]FileEntry) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var dirs, files []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, f := range files {
		rel, _ := filepath.Rel(root, filepath.Join(dir, f.Name()))
		*result = append(*result, FileEntry{RelPath: filepath.ToSlash(rel), Indent: depth})
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	for _, d := range dirs {
		subDir := filepath.Join(dir, d.Name())
		var sub []FileEntry
		if err := walkDir(root, subDir, depth+1, &sub); err != nil {
			return err
		}
		if len(sub) > 0 {
			rel, _ := filepath.Rel(root, subDir)
			*result = append(*result, FileEntry{IsDir: true, DirLabel: filepath.ToSlash(rel), Indent: depth})
			*result = append(*result, sub...)
		}
	}
	return nil
}

// EntryContains reports whether entries contains a file with the given relative path.
func EntryContains(entries []FileEntry, relPath string) bool {
	for _, e := range entries {
		if !e.IsDir && e.RelPath == relPath {
			return true
		}
	}
	return false
}

// FileIcon returns an emoji icon for a file based on its extension.
func FileIcon(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".md"):
		return "📄"
	case strings.HasSuffix(lower, ".json"):
		return "📋"
	case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "⚙️"
	case strings.HasSuffix(lower, ".sh"):
		return "⚡"
	case strings.HasSuffix(lower, ".txt"):
		return "📝"
	default:
		return "📃"
	}
}

// htmlTmpl is the single-page HTML template for the skill web viewer.
// Uses marked.js + highlight.js (CDN) for rendering.
const htmlTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.SkillName}} — Skill Viewer</title>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css">
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root {
    --bg:        #0d1117;
    --sidebar-bg:#161b22;
    --border:    #30363d;
    --text:      #c9d1d9;
    --text-dim:  #8b949e;
    --accent:    #58a6ff;
    --active-bg: #1f2937;
    --active-fg: #58a6ff;
    --code-bg:   #161b22;
    --hover-bg:  #1c2128;
  }
  html, body { height: 100%; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
  body { display: flex; background: var(--bg); color: var(--text); }

  /* ── Sidebar ── */
  #sidebar {
    width: 240px;
    min-width: 180px;
    display: flex;
    flex-direction: column;
    background: var(--sidebar-bg);
    border-right: 1px solid var(--border);
    overflow: hidden;
  }
  #sidebar-header {
    padding: 16px 14px 12px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  #sidebar-header .skill-id {
    font-size: 11px;
    color: var(--text-dim);
    text-transform: uppercase;
    letter-spacing: .05em;
    margin-bottom: 4px;
  }
  #sidebar-header .skill-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
    word-break: break-word;
  }
  #file-list { flex: 1; overflow-y: auto; padding: 6px 0; }
  #file-list li { list-style: none; }
  /* directory label row */
  #file-list li.dir-label {
    padding: 8px 14px 4px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-dim);
    letter-spacing: .04em;
    text-transform: uppercase;
    display: flex;
    align-items: center;
    gap: 5px;
  }
  /* file link row */
  #file-list li a {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 14px;
    font-size: 13px;
    color: var(--text-dim);
    cursor: pointer;
    text-decoration: none;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    border-left: 2px solid transparent;
    transition: background .1s, color .1s;
  }
  #file-list li a:hover { background: var(--hover-bg); color: var(--text); }
  #file-list li a.active {
    background: var(--active-bg);
    color: var(--active-fg);
    border-left-color: var(--accent);
  }
  #file-list li a .icon { flex-shrink: 0; }

  /* ── Main content ── */
  #main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
  #toolbar {
    display: flex;
    align-items: center;
    padding: 10px 20px;
    border-bottom: 1px solid var(--border);
    background: var(--sidebar-bg);
    font-size: 13px;
    color: var(--text-dim);
    min-height: 44px;
    flex-shrink: 0;
  }
  #toolbar #current-file { color: var(--text); font-weight: 500; }
  #content { flex: 1; overflow-y: auto; padding: 32px 48px; }

  /* ── Markdown styles ── */
  #md-body { max-width: 1200px; width: 100%; margin: 0 auto; line-height: 1.75; font-size: 15px; }
  #md-body h1 { font-size: 2em; border-bottom: 1px solid var(--border); padding-bottom: .3em; margin: 0 0 1em; }
  #md-body h2 { font-size: 1.5em; border-bottom: 1px solid var(--border); padding-bottom: .3em; margin: 1.5em 0 .8em; }
  #md-body h3 { font-size: 1.25em; margin: 1.4em 0 .6em; }
  #md-body h4, #md-body h5, #md-body h6 { margin: 1em 0 .5em; }
  #md-body p { margin-bottom: 1em; }
  #md-body a { color: var(--accent); text-decoration: none; }
  #md-body a:hover { text-decoration: underline; }
  #md-body code {
    background: var(--code-bg);
    border: 1px solid var(--border);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: "SFMono-Regular", Consolas, monospace;
    font-size: .875em;
  }
  #md-body pre {
    background: var(--code-bg) !important;
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    overflow-x: auto;
    margin-bottom: 1.2em;
  }
  #md-body pre code { background: none; border: none; padding: 0; font-size: .875em; }
  #md-body blockquote {
    border-left: 4px solid var(--border);
    padding: 4px 16px;
    color: var(--text-dim);
    margin: 0 0 1em;
  }
  #md-body ul, #md-body ol { padding-left: 1.8em; margin-bottom: 1em; }
  #md-body li { margin-bottom: .3em; }
  #md-body table { border-collapse: collapse; width: 100%; margin-bottom: 1.2em; font-size: 14px; }
  #md-body th, #md-body td { border: 1px solid var(--border); padding: 8px 14px; text-align: left; }
  #md-body th { background: var(--code-bg); font-weight: 600; }
  #md-body tr:nth-child(even) { background: rgba(255,255,255,.02); }
  #md-body img { max-width: 100%; border-radius: 6px; }
  #md-body hr { border: none; border-top: 1px solid var(--border); margin: 1.5em 0; }

  /* ── Raw text ── */
  #raw-body {
    white-space: pre-wrap;
    word-break: break-all;
    font-family: "SFMono-Regular", Consolas, monospace;
    font-size: 13px;
    line-height: 1.6;
    color: var(--text);
  }

  /* ── Status ── */
  .msg { color: var(--text-dim); font-size: 14px; padding: 20px 0; }
  .err { color: #f85149; }

  /* ── Front matter card ── */
  .fm-card {
    background: var(--code-bg);
    margin-bottom: 2em;
    font-size: 13px;
  }
  .fm-table { border-collapse: collapse; width: 100%; margin-bottom: 0; }
  .fm-table tr { border-bottom: 1px solid var(--border); }
  .fm-table tr:last-child { border-bottom: none; }
  .fm-key {
    padding: 7px 14px;
    color: var(--text-dim);
    white-space: nowrap;
    width: 130px;
    font-weight: 500;
  }
  .fm-val { padding: 7px 14px; color: var(--text); }
  .fm-tag {
    display: inline-block;
    background: rgba(88,166,255,.12);
    color: var(--accent);
    border: 1px solid rgba(88,166,255,.25);
    border-radius: 4px;
    padding: 1px 7px;
    font-size: 12px;
    margin: 1px 2px;
  }
  ::-webkit-scrollbar { width: 6px; height: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
</style>
</head>
<body data-default="{{.DefaultFile}}">

<nav id="sidebar">
  <div id="sidebar-header">
    <div class="skill-id">{{.SkillID}}</div>
    <div class="skill-name">{{.SkillName}}</div>
  </div>
  <ul id="file-list">
    {{- range .Entries}}
    {{- if .IsDir}}
    <li class="dir-label" style="padding-left: {{indent .Indent}}px">
      <span>📁</span>{{.DirLabel}}
    </li>
    {{- else}}
    <li><a href="#" data-file="{{.RelPath}}" onclick="loadFile(this);return false;"
         style="padding-left: {{indentFile .Indent}}px">
      <span class="icon">{{skillFileIcon .RelPath}}</span>{{fileName .RelPath}}
    </a></li>
    {{- end}}
    {{- end}}
  </ul>
</nav>

<div id="main">
  <div id="toolbar">
    <span id="current-file">—</span>
  </div>
  <div id="content">
    <div id="md-body"></div>
    <pre id="raw-body" style="display:none"></pre>
    <div id="msg" class="msg">Loading…</div>
  </div>
</div>

<script src="https://cdnjs.cloudflare.com/ajax/libs/marked/9.1.6/marked.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
<script>
  marked.use({
    renderer: (function() {
      const r = new marked.Renderer();
      r.code = function(code, lang) {
        if (lang && hljs.getLanguage(lang)) {
          return '<pre><code class="hljs">' + hljs.highlight(code, { language: lang }).value + '</code></pre>';
        }
        return '<pre><code class="hljs">' + hljs.highlightAuto(code).value + '</code></pre>';
      };
      return r;
    })(),
    breaks: true,
    gfm: true,
  });

  // parseFrontMatter splits content into {meta: {}, body: "..."}.
  // Expects "---\nkey: val\n---\n..." format. Returns null meta if not found.
  function parseFrontMatter(content) {
    const m = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
    if (!m) return { meta: null, body: content };
    const meta = {};
    for (const line of m[1].split(/\r?\n/)) {
      const colon = line.indexOf(':');
      if (colon < 0) continue;
      const key = line.slice(0, colon).trim();
      let val = line.slice(colon + 1).trim();
      // Handle simple inline list: [a, b, c]
      if (val.startsWith('[') && val.endsWith(']')) {
        val = val.slice(1, -1).split(',').map(s => s.trim()).filter(Boolean);
      }
      if (key) meta[key] = val;
    }
    return { meta, body: m[2] };
  }

  // renderFrontMatter renders the YAML meta as a styled info card HTML string.
  function renderFrontMatter(meta) {
    const LABELS = {
      id: 'ID', name: 'Name', description: 'Description',
      version: 'Version', supported_agents: 'Agents', install_entry: 'Install Entry',
    };
    const order = ['id', 'name', 'description', 'version', 'supported_agents', 'install_entry'];
    const keys = [...order.filter(k => meta[k] !== undefined),
                  ...Object.keys(meta).filter(k => !order.includes(k))];
    let rows = '';
    for (const k of keys) {
      const v = meta[k];
      const label = LABELS[k] || k;
      let cell;
      if (Array.isArray(v)) {
        cell = v.map(s => '<code class="fm-tag">' + esc(s) + '</code>').join(' ');
      } else {
        cell = '<span>' + esc(String(v)) + '</span>';
      }
      rows += '<tr><td class="fm-key">' + esc(label) + '</td><td class="fm-val">' + cell + '</td></tr>';
    }
    return '<div class="fm-card"><table class="fm-table">' + rows + '</table></div>';
  }

  function esc(s) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  }

  function renderContent(content, isMarkdown) {
    if (!isMarkdown) return null; // handled separately
    const { meta, body } = parseFrontMatter(content);
    let html = '';
    if (meta) html += renderFrontMatter(meta);
    html += marked.parse(body);
    return html;
  }

  // currentDir tracks the directory of the currently viewed file for resolving relative links.
  let currentDir = '';

  function loadFile(el) {
    document.querySelectorAll('#file-list a').forEach(a => a.classList.remove('active'));
    el.classList.add('active');

    const relPath = el.getAttribute('data-file');
    loadPath(relPath);
  }

  // loadPath loads a file by its relative path (from skill root), resolving relative links.
  function loadPath(relPath) {
    // Normalize: strip leading "./"
    relPath = relPath.replace(/^\.\//, '');

    // Update current directory for resolving future relative links
    const slash = relPath.lastIndexOf('/');
    currentDir = slash >= 0 ? relPath.substring(0, slash) : '';

    // Sync sidebar highlight
    document.querySelectorAll('#file-list a').forEach(a => a.classList.remove('active'));
    const sidebarLink = document.querySelector('#file-list a[data-file="' + CSS.escape(relPath) + '"]');
    if (sidebarLink) sidebarLink.classList.add('active');

    document.getElementById('current-file').textContent = relPath;
    document.getElementById('msg').textContent = 'Loading\u2026';
    document.getElementById('msg').className = 'msg';
    document.getElementById('msg').style.display = '';
    document.getElementById('md-body').style.display = 'none';
    document.getElementById('raw-body').style.display = 'none';

    fetch('/file?name=' + encodeURIComponent(relPath))
      .then(r => r.json())
      .then(data => {
        document.getElementById('msg').style.display = 'none';
        if (data.error) {
          document.getElementById('msg').textContent = 'Error: ' + data.error;
          document.getElementById('msg').className = 'msg err';
          document.getElementById('msg').style.display = '';
          return;
        }
        if (data.isMarkdown) {
          document.getElementById('md-body').innerHTML = renderContent(data.content, true);
          document.getElementById('md-body').style.display = '';
          document.getElementById('raw-body').style.display = 'none';
          // Intercept relative links inside the rendered markdown
          interceptMarkdownLinks();
        } else {
          document.getElementById('raw-body').textContent = data.content;
          document.getElementById('raw-body').style.display = '';
          document.getElementById('md-body').style.display = 'none';
        }
      })
      .catch(err => {
        document.getElementById('msg').textContent = 'Network error: ' + err.message;
        document.getElementById('msg').className = 'msg err';
        document.getElementById('msg').style.display = '';
      });
  }

  // interceptMarkdownLinks rewrites relative links in the rendered markdown
  // so they load via our viewer instead of navigating the browser.
  function interceptMarkdownLinks() {
    document.querySelectorAll('#md-body a').forEach(a => {
      const href = a.getAttribute('href');
      if (!href) return;
      // Skip absolute URLs and anchor-only links
      if (/^https?:\/\//i.test(href) || href.startsWith('#')) return;

      a.addEventListener('click', function(e) {
        e.preventDefault();
        // Resolve relative href against currentDir
        let target = href;
        if (!target.startsWith('/')) {
          target = currentDir ? currentDir + '/' + target : target;
        } else {
          target = target.replace(/^\//, '');
        }
        // Normalize: collapse "foo/bar/../baz" → "foo/baz"
        target = normalizePath(target);
        loadPath(target);
      });
    });
  }

  // normalizePath collapses ".." segments in a forward-slash path.
  function normalizePath(p) {
    const parts = p.split('/');
    const out = [];
    for (const seg of parts) {
      if (seg === '..') { out.pop(); }
      else if (seg !== '.') { out.push(seg); }
    }
    return out.join('/');
  }

  window.addEventListener('DOMContentLoaded', function() {
    const def = document.body.dataset.default;
    const link = def
      ? document.querySelector('#file-list a[data-file="' + CSS.escape(def) + '"]')
      : document.querySelector('#file-list a');
    if (link) loadFile(link);
    else document.getElementById('msg').textContent = 'No files found.';
  });
</script>
</body>
</html>`
