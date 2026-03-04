package main

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	//go:embed docs/swagger.json
	swaggerDocJSON []byte

	//go:embed docs/swagger.yaml
	swaggerDocYAML []byte
)

const swaggerIndexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Xiaohongshu MCP Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/swagger/doc.json",
        dom_id: "#swagger-ui",
      });
    };
  </script>
</body>
</html>
`

func swaggerRedirectHandler(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
}

func swaggerIndexHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerIndexHTML))
}

func swaggerDocJSONHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", swaggerDocJSON)
}

func swaggerDocYAMLHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/x-yaml; charset=utf-8", swaggerDocYAML)
}
