# seal

Some kind of experimental headless filesystem-based content management system.

* File without extension: registers a handler (maximum one per directory)
  * Default handler: render template
* File with extension: is converted to html, then parsed as a template
* Directory with a plain name: becomes a part of the route
* Directory with a `{parameter}` name: registers a middleware (maximum one per parent directory)
  * Difference to handler: execution continues after middleware returns. Don't want that? Use a handler instead.

## Notes

* Embed images etc. via their filesystem location: use `/{lang}/img.jpg`, not `/de/img.jpg`
