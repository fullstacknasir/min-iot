package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	OpenAPIFilePath = "docs/openapi.yaml"
	AltOpenAPIPath  = "../docs/openapi.yaml"
)

// ServeOpenAPISpec serves the raw OpenAPI specification YAML file.
func ServeOpenAPISpec(c *gin.Context) {
	data, err := os.ReadFile(OpenAPIFilePath)
	if err != nil {
		data, err = os.ReadFile(AltOpenAPIPath)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OpenAPI specification file not found"})
		return
	}

	c.Data(http.StatusOK, "application/yaml", data)
}

// ServeDocsUI serves an interactive Swagger UI explorer pointing to the OpenAPI specification.
func ServeDocsUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>minIoT Platform API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    body { margin: 0; background: #fafafa; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/docs/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
