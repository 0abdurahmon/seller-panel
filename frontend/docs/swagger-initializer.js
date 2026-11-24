window.ui = SwaggerUIBundle({
  url: 'spec/openapi.yaml',
  dom_id: '#swagger-ui',
  deepLinking: true,
  persistAuthorization: true,
  presets: [SwaggerUIBundle.presets.apis],
})
