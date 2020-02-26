/*
Package httpasset lets you embed files in your go binary by simply appending a
zip file to the binary.

Call httpasset.Init("fallbackdir") to get a handle to a http.FileSystem  that
serves files from the appended zip file or a local fallback path.

Example:

	package main

	import (
		"log"
		"net/http"
		"github.com/mjl-/httpasset"
	)

	var httpFS = httpasset.Init("assets")

	func main() {
		http.Handle("/", http.FileServer(httpFS))
		addr := ":8000"
		log.Println("listening on", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}


Build your program, let's say the result is "mybinary".
Now create a zip file, eg on unix:

	(cd assets && zip -rq0 ../assets.zip .)

Append it to the binary:

	cat assets.zip >>mybinary

If you run mybinary, it will serve http on port 8000, serving the files from the
zip file that was appended to the binary.

Net/http's FileServer will redirect requests for /index.html to /, and
handle requests for / by returning the file /index.html if it exists.  If
/index.html doesn't exist, it will list the contents of the directory. For
net/http's Dir(), that works (the fallback `fs` in the example code). For
httpasset's `fs`, reading directories isn't supported and returns an empty list
of files. Listing files is often not needed, simpler, and it's usually better
not to leak such information.

Net/http's FileServer also supports requests for random i/o (range requests),
and advertises this in response headers.  Files in zip files can be compressed.
Compressed files don't support random access. Httpasset returns an error when
asked to serve range requests for compressed files. It's recommeded to add files
to the zip file uncompressed. The -0 flag takes care of this in the example
given earlier.

To make this work, an assumption about zip files is made: That the central
directory (with a list of files inside the zip file) comes right before the "end
of central directory" marker.  This is almost always the case with zip files.
With this assumption, httpasset can locate the start and end of the zip file
that is appended to the binary, which archive/zip needs in order to parse the
zip file.

Some existing tools for reading zip files can still read the binary-with-zipfile
as a zip file.  For example 7z, and the unzip command-line tool.  Windows XP's
explorer zip opener does NOT seem to understand it, and also Mac OS X's archive
utility gets confused.

This has been tested with binaries on Linux (Ubuntu 12.04), Mac OS X (10.9.2)
and Windows 8.1.  These operating systems don't seem to mind extra data at the
end of the binary.
*/
package httpasset
