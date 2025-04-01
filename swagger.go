package swagger

import (
	"encoding/json"
	"fmt"
	"github.com/darwinOrg/go-common/utils"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"github.com/go-openapi/spec"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	contentTypeJson = "application/json"
)

type ExportSwaggerRequest struct {
	Title       string
	Description string
	Version     string
	RequestApis []*wrapper.RequestApi
	OutDir      string
	ServiceName string
}

func ExposeGinSwagger(e *gin.Engine) {
	swaggerProps := buildSwaggerProps(&ExportSwaggerRequest{
		RequestApis: wrapper.GetRequestApis(),
	})
	swaggerInfo := &swag.Spec{
		InfoInstanceName: "swagger",
		SwaggerTemplate:  utils.MustConvertBeanToJsonString(swaggerProps),
		LeftDelim:        "{{",
		RightDelim:       "}}",
	}
	swag.Register(swaggerInfo.InstanceName(), swaggerInfo)
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func ExportSwaggerFile(req *ExportSwaggerRequest) string {
	if len(req.RequestApis) == 0 {
		log.Print("没有需要导出的接口定义")
		return ""
	}
	if req.ServiceName == "" {
		req.ServiceName = "service"
	}

	swaggerProps := buildSwaggerProps(req)
	filename := fmt.Sprintf("%s/%s.swagger.json", req.OutDir, req.ServiceName)
	saveToFile(swaggerProps, filename)
	return filename
}

func CreateSchemaForObject(obj any) *spec.Schema {
	return createSchemaForType(reflect.TypeOf(obj), 0)
}

func buildSwaggerProps(req *ExportSwaggerRequest) spec.SwaggerProps {
	if req.Title == "" {
		req.Title = "接口文档"
	}
	if req.Description == "" {
		req.Description = "接口描述"
	}
	if req.OutDir == "" {
		req.OutDir = "openapi/v1"
	}
	if req.Version == "" {
		req.Version = "v1.0.0"
	}

	return spec.SwaggerProps{
		Swagger:             "2.0",
		Definitions:         spec.Definitions{},
		SecurityDefinitions: spec.SecurityDefinitions{},
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:       req.Title,
				Description: req.Description,
				Version:     req.Version,
			},
		},
		Paths: buildApiPaths(req.RequestApis),
	}
}

func buildApiPaths(requestApis []*wrapper.RequestApi) *spec.Paths {
	paths := map[string]spec.PathItem{}

	for _, api := range requestApis {
		url := fmt.Sprintf("%s/%s", api.BasePath, api.RelativePath)
		url = strings.ReplaceAll(url, "//", "/")

		var parameters []spec.Parameter
		if api.Method == http.MethodGet {
			parameters = buildGetParameters(api)
		} else {
			parameters = buildPostParameters(api)
		}

		operation := &spec.Operation{
			OperationProps: spec.OperationProps{
				Summary:     api.Remark,
				Description: api.Remark,
				Consumes:    []string{contentTypeJson},
				Produces:    []string{contentTypeJson},
				Parameters:  parameters,
				Responses:   buildResponses(api),
			},
		}

		itemProps := spec.PathItemProps{}
		if api.Method == http.MethodGet {
			itemProps.Get = operation
		} else {
			itemProps.Post = operation
		}

		paths[url] = spec.PathItem{
			PathItemProps: itemProps,
		}
	}

	return &spec.Paths{
		Paths: paths,
	}
}

func buildGetParameters(api *wrapper.RequestApi) []spec.Parameter {
	tpe := reflect.TypeOf(api.RequestObject)
	for tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
	}
	cnt := tpe.NumField()
	var parameters []spec.Parameter

	for i := 0; i < cnt; i++ {
		field := tpe.Field(i)
		p := *spec.QueryParam(extractNameFromField(field))

		switch field.Type.Kind() {
		case reflect.String:
			p.Type = "string"
		case reflect.Bool:
			p.Type = "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			p.Type = "integer"
		case reflect.Float32, reflect.Float64:
			p.Type = "number"
		case reflect.Slice, reflect.Array:
			p.Type = "array"
		case reflect.Map:
			continue
		default:
			fmt.Printf("Unsupported field type: %s\n", field.Type.Kind())
		}

		p.Required = extractRequiredFlagFromField(field)
		p.Description = extractDescriptionFromField(field)

		parameters = append(parameters, p)
	}

	return parameters
}

func buildPostParameters(api *wrapper.RequestApi) []spec.Parameter {
	schema := CreateSchemaForObject(api.RequestObject)
	bodyParam := *spec.BodyParam("body", schema)
	bodyParam.Required = true
	return []spec.Parameter{bodyParam}
}

func createSchemaForType(tpe reflect.Type, depth int) *spec.Schema {
	// 限制递归深度最大为8
	if depth > 8 {
		return nil
	}

	for tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
	}

	schema := &spec.Schema{}
	switch tpe.Kind() {
	case reflect.String:
		schema.Type = []string{"string"}
	case reflect.Bool:
		schema.Type = []string{"boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = []string{"integer"}
	case reflect.Float32, reflect.Float64:
		schema.Type = []string{"number"}
	case reflect.Slice, reflect.Array:
		elemType := tpe.Elem()
		itemSchema := createSchemaForType(elemType, depth+1)
		schema.Type = []string{"array"}
		schema.Items = &spec.SchemaOrArray{Schema: itemSchema}
	case reflect.Map:
		keyType := tpe.Key()
		if keyType.Kind() != reflect.String {
			panic("Map keys must be strings in OpenAPI schemas.")
		}
		valueType := tpe.Elem()
		valueSchema := createSchemaForType(valueType, depth+1)
		schema.Type = []string{"object"}
		schema.AdditionalProperties = &spec.SchemaOrBool{
			Allows: true,
			Schema: valueSchema,
		}
	case reflect.Struct:
		schema.Properties = make(map[string]spec.Schema)
		schema.Required = make([]string, 0)
		cnt := tpe.NumField()

		for i := 0; i < cnt; i++ {
			field := tpe.Field(i)

			if strings.Contains(tpe.String(), "result.Result") && field.Name == "Data" {
				rt := reflect.New(tpe).Elem().Interface()
				dataType := reflect.ValueOf(rt).Field(i).Type()
				for dataType.Kind() == reflect.Pointer {
					dataType = dataType.Elem()
				}
				field.Type = dataType
			}

			tpeStr := tpe.String()
			tpeStr = strings.TrimPrefix(tpeStr, "*")

			fieldTypeStr := field.Type.String()
			fieldTypeStr = strings.TrimPrefix(fieldTypeStr, "[]")
			fieldTypeStr = strings.TrimPrefix(fieldTypeStr, "*")

			var property *spec.Schema

			// 如果有结构体类型名称和字段名称相同，则不再递归，以免无限递归
			if tpeStr == fieldTypeStr {
				property = &spec.Schema{}

				switch field.Type.Kind() {
				case reflect.Slice, reflect.Array:
					property.Type = []string{"array"}
				default:
					property.Type = []string{"object"}
				}
			} else {
				property = createSchemaForType(field.Type, depth+1)
				if property == nil {
					continue
				}
			}

			property.Title = extractTitleFromField(field)
			property.Description = extractDescriptionFromField(field)
			fieldName := extractNameFromField(field)
			schema.Properties[fieldName] = *property

			if extractRequiredFlagFromField(field) {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	default:
		fmt.Printf("Unsupported field type: %s\n", tpe.Kind())
	}

	return schema
}

func buildResponses(api *wrapper.RequestApi) *spec.Responses {
	return &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: map[int]spec.Response{
				http.StatusOK: {
					ResponseProps: spec.ResponseProps{
						Description: "成功",
						Schema:      CreateSchemaForObject(api.ResponseObject),
					},
				},
			},
		},
	}
}

func extractNameFromField(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		return strings.ReplaceAll(jsonTag, ",omitempty", "")
	} else {
		if len(field.Name) == 1 {
			return strings.ToLower(field.Name)
		}

		return strings.ToLower(field.Name[0:1]) + field.Name[1:]
	}
}

func extractTitleFromField(field reflect.StructField) string {
	title := field.Tag.Get("title")
	if title != "" {
		return title
	} else {
		return extractDescriptionFromField(field)
	}
}

func extractRequiredFlagFromField(field reflect.StructField) bool {
	bindingTag := field.Tag.Get("binding")
	return bindingTag != "" && strings.Contains(bindingTag, "required")
}

func extractDescriptionFromField(field reflect.StructField) string {
	return field.Tag.Get("remark")
}

func saveToFile(swaggerProps spec.SwaggerProps, filename string) {
	swaggerJSON, err := json.MarshalIndent(swaggerProps, "", "  ")
	if err != nil {
		panic(err)
	}

	dirPath := filepath.Dir(filename)
	if err = os.MkdirAll(dirPath, os.ModePerm); err != nil {
		panic(err)
	}

	_, err = os.Create(filename)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filename, swaggerJSON, os.ModePerm)
	if err != nil {
		panic(err)
	}
}
