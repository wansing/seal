package seal

// A ContentFunc processes file content, e. g. by populating dir.Template.New(filestem).
type ContentFunc func(dir *Dir, filestem string, filecontent []byte) error
