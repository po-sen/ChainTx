package dto

type GetOpenAPISpecQuery struct{}

type OpenAPISpecOutput struct {
	Content     []byte
	ContentType string
}
