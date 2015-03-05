# Polaroid:	Creates a clone of a given URL with all of the external resources remapped to use global URLs. It is designed for use as a self contained phish endpoint generator and web server.
## Usage:	<polaroid> <-t URL> [...]

### Opt:	-t <URL>	Sets the target URL
### Opt:	-s			Start a web server and serve the modified page, then when input is entered, gather the data and retransmit the request to the original page
### Opt:	-e <file>	Embed a java payload in the HTML
### Opt:	-c <class>	Specify entry class for java embed object

