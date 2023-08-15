# seal

Some kind of experimental headless filesystem-based content management system.

* Directory: is matched to request uri
* File without extension: registers a handler (maximum one per directory)
  * handler is always run
  * handler can return true in order to skip itself
  * default handler executes template if the remaining path is empty
* File with extension: is converted to html, then parsed as a template
