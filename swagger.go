package swagger

import (
	"encoding/json"
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const (
	contentTypeJson     = "application/json"
	apifoxImportDataUrl = "https://api.apifox.com/api/v1/projects/%s/import-data?locale=zh-CN"
	apifoxCreateDirUrl  = "https://api.apifox.com/api/v1/projects/%s/api-folders"
)

type ExportSwaggerRequest struct {
	ServiceName string
	Title       string
	Description string
	OutDir      string
	Version     string
	RequestApis []*wrapper.RequestApi
}

type SyncToApifoxRequest struct {
	*ExportSwaggerRequest
	ProjectId           string              `json:"projectId"`           // 项目 ID，打开 Apifox 进入项目里的“项目设置”查看
	AccessToken         string              `json:"accessToken"`         // 身份认证，https://apifox.com/help/openapi/
	ApiOverwriteMode    ApiOverwriteMode    `json:"apiOverwriteMode"`    // 匹配到相同接口时的覆盖模式，不传表示忽略
	SchemaOverwriteMode SchemaOverwriteMode `json:"schemaOverwriteMode"` // 匹配到相同数据模型时的覆盖模式，不传表示忽略
	SyncApiFolder       bool                `json:"syncApiFolder"`       // 是否同步更新接口所在目录
	ApiFolderId         int64               `json:"apiFolderId"`         // 导入到目标目录的ID，不传表示导入到根目录
	ImportBasePath      bool                `json:"importBasePath"`      // 是否在接口路径加上basePath，建议不传，即为false，推荐将BasePath放到环境里的“前置URL”里
}

type apifoxImportDataBody struct {
	ImportFormat        string              `json:"importFormat"`
	Data                string              `json:"data"`
	ApiOverwriteMode    ApiOverwriteMode    `json:"apiOverwriteMode"`
	SchemaOverwriteMode SchemaOverwriteMode `json:"schemaOverwriteMode"`
	SyncApiFolder       bool                `json:"syncApiFolder"`
	ApiFolderId         *string             `json:"apiFolderId,omitempty"`
	ImportBasePath      bool                `json:"importBasePath"`
}

type apifoxResult[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type apifoxCreateDirData struct {
	Id int64 `json:"id"`
}

type ApiOverwriteMode string

const (
	ApiOverwriteModeMethodAndPath ApiOverwriteMode = "methodAndPath"
	ApiOverwriteModeBoth          ApiOverwriteMode = "both"
	ApiOverwriteModeMerge         ApiOverwriteMode = "merge"
	ApiOverwriteModeIgnore        ApiOverwriteMode = "ignore"
)

type SchemaOverwriteMode string

const (
	SchemaOverwriteModeName   SchemaOverwriteMode = "name"
	SchemaOverwriteModeBoth   SchemaOverwriteMode = "both"
	SchemaOverwriteModeMerge  SchemaOverwriteMode = "merge"
	SchemaOverwriteModeIgnore SchemaOverwriteMode = "ignore"
)

func ExportSwaggerFile(req *ExportSwaggerRequest) {
	if len(req.RequestApis) == 0 {
		panic("没有需要导出的接口定义")
	}
	if req.ServiceName == "" {
		panic("服务名不能为空")
	}

	swaggerProps := buildSwaggerProps(req)
	filename := fmt.Sprintf("%s/%s.swagger.json", req.OutDir, req.ServiceName)
	saveToFile(swaggerProps, filename)
}

func SyncSwaggerToApifox(req *SyncToApifoxRequest) {
	if len(req.RequestApis) == 0 {
		panic("没有需要导出的接口定义")
	}
	if req.OutDir == "" {
		panic("同步目录不能为空")
	}

	swaggerProps := buildSwaggerProps(req.ExportSwaggerRequest)
	syncToApifox(swaggerProps, req)
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
	schema := createSchemaForType(reflect.TypeOf(api.RequestObject))
	bodyParam := *spec.BodyParam("body", schema)
	bodyParam.Required = true
	return []spec.Parameter{bodyParam}
}

func createSchemaForType(tpe reflect.Type) *spec.Schema {
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
		itemSchema := createSchemaForType(elemType)
		schema.Type = []string{"array"}
		schema.Items = &spec.SchemaOrArray{Schema: itemSchema}
	case reflect.Map:
		keyType := tpe.Key()
		if keyType.Kind() != reflect.String {
			panic("Map keys must be strings in OpenAPI schemas.")
		}
		valueType := tpe.Elem()
		valueSchema := createSchemaForType(valueType)
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

			property := createSchemaForType(field.Type)
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
						Schema:      createSchemaForType(reflect.TypeOf(api.ResponseObject)),
					},
				},
			},
		},
	}
}

func extractNameFromField(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		return jsonTag
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

func syncToApifox(swaggerProps spec.SwaggerProps, req *SyncToApifoxRequest) {
	swaggerJSON, err := json.MarshalIndent(swaggerProps, "", "  ")
	if err != nil {
		panic(err)
	}

	if string(req.ApiOverwriteMode) == "" {
		req.ApiOverwriteMode = ApiOverwriteModeIgnore
	}
	if string(req.SchemaOverwriteMode) == "" {
		req.SchemaOverwriteMode = SchemaOverwriteModeIgnore
	}

	importDataUrl := fmt.Sprintf(apifoxImportDataUrl, req.ProjectId)

	headers := map[string]string{
		"X-Apifox-Version": "2024-01-20",
		"Authorization":    "Bearer " + req.AccessToken,
	}

	importDataBody := apifoxImportDataBody{
		ImportFormat:        "openapi",
		Data:                string(swaggerJSON),
		ApiOverwriteMode:    req.ApiOverwriteMode,
		SchemaOverwriteMode: req.SchemaOverwriteMode,
		SyncApiFolder:       req.SyncApiFolder,
		ImportBasePath:      req.ImportBasePath,
	}

	ctx := &dgctx.DgContext{TraceId: uuid.NewString()}

	if req.ApiFolderId > 0 {
		apiFolderId := strconv.FormatInt(req.ApiFolderId, 10)
		importDataBody.ApiFolderId = &apiFolderId
	} else if req.OutDir != "" {
		dirs := strings.Split(req.OutDir, "/")
		createDirUrl := fmt.Sprintf(apifoxCreateDirUrl, req.ProjectId)
		dirId := "0"

		for _, dir := range dirs {
			createDirParams := map[string]string{
				"name":     dir,
				"parentId": dirId,
			}

			respBytes, err := dghttp.Client11.DoPostFormUrlEncoded(ctx, createDirUrl, createDirParams, headers)
			if err != nil {
				panic(err)
			}
			dglogger.Infof(ctx, "resp: %s", string(respBytes))
			apifoxResp := utils.MustConvertJsonBytesToBean[apifoxResult[apifoxCreateDirData]](respBytes)
			dirId = strconv.FormatInt(apifoxResp.Data.Id, 10)
		}

		importDataBody.ApiFolderId = &dirId
	}

	_, err = dghttp.Client11.DoPostJson(ctx, importDataUrl, importDataBody, headers)
	if err != nil {
		panic(err)
	}
}
