# seal

Some kind of experimental headless filesystem-based content management system.

* Directory: is matched to request uri
  * Extension: call handler
  * No extension: execute HTML templates
* File: is converted to html, then parsed as a template
