package install

func DetectConflict(plan Plan, exists bool) Conflict {
	mode := plan.ConflictMode
	if mode == "" {
		mode = ConflictModeOverwrite
	}
	return Conflict{
		Exists: exists,
		Mode:   mode,
	}
}
