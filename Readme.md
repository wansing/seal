# seal

Some kind of experimental headless filesystem-based content management system.

* Directory: is matched to request uri
* File without extension: registers a handler (maximum one per directory)
  * handler is always run
  * boolean return value tells whether to continue execution
  * default handler executes template if the remaining path is empty
* File with extension: is converted to html, then parsed as a template

## How to nest repos

```
otherRepo := seal.MakeDirRepository(config, "../other-repo")

rootRepo.Conf.Handlers["other-repo"] = func(dir *seal.Dir, filestem string, filecontent []byte) seal.Handler {
    if err := otherRepo.Update(dir); err != nil {
        log.Printf("error updating other repo: %v", err)
    }
    return otherRepo.Serve
}

http.HandleFunc("/git-update-other", otherRepo.GitUpdateHandler("change-me", srv))
```
