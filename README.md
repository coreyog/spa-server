# spa-server

Serves a directory BUT if a request would normally result in a 404, instead the default document is returned. This allows for Angular or React apps to use more natural routing (i.e. http://localhost/app/dashboard instead of http://localhost/#/app/dashboard).