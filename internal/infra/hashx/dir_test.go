package hashx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestSumDeployedIgnoresRuntimeArtifacts(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill"), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "__pycache__"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "__pycache__", "helper.cpython-312.pyc"), []byte("bytecode"), 0o644))

	first, err := SumDeployed(root)
	assert.NoErr(t, err)

	// 解释器缓存变化不属于技能内容变化
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "__pycache__", "helper.cpython-314.pyc"), []byte("other"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "helper.pyc"), []byte("bytecode"), 0o644))
	second, err := SumDeployed(root)
	assert.NoErr(t, err)
	assert.Eq(t, first, second)

	// 真实内容变化必须被发现
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("changed"), 0o644))
	third, err := SumDeployed(root)
	assert.NoErr(t, err)
	assert.NotEq(t, first, third)
}

func TestSumDeployedDetectsAddedAndRemovedFiles(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "SKILL.md")
	assert.NoErr(t, os.WriteFile(file, []byte("skill"), 0o644))
	first, err := SumDeployed(root)
	assert.NoErr(t, err)

	extra := filepath.Join(root, "references", "schema.md")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(extra), 0o755))
	assert.NoErr(t, os.WriteFile(extra, []byte("schema"), 0o644))
	second, err := SumDeployed(root)
	assert.NoErr(t, err)
	assert.NotEq(t, first, second)

	assert.NoErr(t, os.Remove(extra))
	third, err := SumDeployed(root)
	assert.NoErr(t, err)
	assert.Eq(t, first, third)
}

// SumDir 用于 source 目录扫描，保持原有语义：只忽略 .git。
func TestSumDirKeepsRuntimeArtifacts(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill"), 0o644))
	pycache := filepath.Join(root, "__pycache__")
	assert.NoErr(t, os.MkdirAll(pycache, 0o755))

	first, err := SumDir(root)
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(pycache, "helper.pyc"), []byte("bytecode"), 0o644))
	second, err := SumDir(root)

	assert.NoErr(t, err)
	assert.NotEq(t, first, second)
}
