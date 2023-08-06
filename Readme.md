# seal

Some kind of experimental headless filesystem-based content management system.

* File without extension: registers a handler (maximum one per directory)
  * Default handler: follow route (by directory names), render template if remaining path is empty
  * First handler is executed (routing is left to the handler)
* File with extension: is converted to html, then parsed as a template

## Notes

* Embed images etc. via their filesystem location: use `/{lang}/img.jpg`, not `/de/img.jpg`
