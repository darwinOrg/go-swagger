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
	"log"
	"strconv"
	"strings"
)

const (
	apifoxImportDataUrl    = "https://api.apifox.com/api/v1/projects/%s/import-data?locale=zh-CN"
	apifoxCreateFolderUrl  = "https://api.apifox.com/api/v1/projects/%s/api-folders?locale=zh-CN"
	apifoxDetailFoldersUrl = "https://api.apifox.com/api/v1/projects/%s/api-detail-folders?locale=zh-CN"
)

type SyncToApifoxRequest struct {
	Title               string
	Description         string
	Version             string
	RequestApis         []*wrapper.RequestApi
	ProjectId           string              // 项目 ID，打开 Apifox 进入项目里的“项目设置”查看
	AccessToken         string              // 身份认证，https://apifox.com/help/openapi/
	ApiOverwriteMode    ApiOverwriteMode    // 匹配到相同接口时的覆盖模式，不传表示忽略
	SchemaOverwriteMode SchemaOverwriteMode // 匹配到相同数据模型时的覆盖模式，不传表示忽略
	SyncApiFolder       bool                // 是否同步更新接口所在目录
	ImportBasePath      bool                // 是否在接口路径加上basePath，建议不传，即为false，推荐将BasePath放到环境里的“前置URL”里
	ApiFolderPath       string              // 导入的目标目录路径，多级目录用“/”分割，若目录不存在则自动创建
}

type apifoxImportDataBody struct {
	ImportFormat        string              `json:"importFormat"`
	Data                string              `json:"data"`
	ApiOverwriteMode    ApiOverwriteMode    `json:"apiOverwriteMode"`
	SchemaOverwriteMode SchemaOverwriteMode `json:"schemaOverwriteMode"`
	SyncApiFolder       bool                `json:"syncApiFolder"`
	ApiFolderId         string              `json:"apiFolderId,omitempty"`
	ImportBasePath      bool                `json:"importBasePath"`
}

type apifoxResult[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type apifoxCreateDirData struct {
	Id int64 `json:"id"`
}

type apifoxDetailFoldersData struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	ParentId string `json:"parentId"`
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

func SyncSwaggerToApifox(req *SyncToApifoxRequest) {
	if len(req.RequestApis) == 0 {
		log.Print("没有需要导出的接口定义")
		return
	}

	swaggerProps := buildSwaggerProps(&ExportSwaggerRequest{
		Title:       req.Title,
		Description: req.Description,
		Version:     req.Version,
		RequestApis: req.RequestApis,
	})

	syncToApifox(swaggerProps, req)
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
	var apiFolderId int64

	if req.ApiFolderPath != "" {
		folders := strings.Split(req.ApiFolderPath, "/")
		createFolderUrl := fmt.Sprintf(apifoxCreateFolderUrl, req.ProjectId)
		detailFoldersUrl := fmt.Sprintf(apifoxDetailFoldersUrl, req.ProjectId)
		detailFoldersRespBytes, err := dghttp.Client11.DoGet(ctx, detailFoldersUrl, nil, headers)
		if err != nil {
			panic(err)
		}
		dglogger.Infof(ctx, "detailFoldersRespBytes: %s", string(detailFoldersRespBytes))
		detailFoldersResp := utils.MustConvertJsonBytesToBean[apifoxResult[[]*apifoxDetailFoldersData]](detailFoldersRespBytes)
		if !detailFoldersResp.Success {
			panic("调用apifox获取目录详情列表接口失败")
		}
		folderDatas := detailFoldersResp.Data

		foldersSize := len(folders)
		for i := 0; i < foldersSize; i++ {
			folderName := folders[i]
			found := false

			for _, folderData := range folderDatas {
				if folderData.Name == folderName {
					apiFolderId = folderData.Id
					found = true
					break
				}
			}

			if !found {
				createFolderParams := map[string]string{
					"name":     folderName,
					"parentId": strconv.FormatInt(apiFolderId, 10),
				}

				createFolderRespBytes, err := dghttp.Client11.DoPostFormUrlEncoded(ctx, createFolderUrl, createFolderParams, headers)
				if err != nil {
					panic(err)
				}
				dglogger.Infof(ctx, "createFolderResp: %s", string(createFolderRespBytes))
				createFolderRespResp := utils.MustConvertJsonBytesToBean[apifoxResult[apifoxCreateDirData]](createFolderRespBytes)
				if !createFolderRespResp.Success {
					panic("调用apifox创建目录接口失败")
				}
				apiFolderId = createFolderRespResp.Data.Id
			}
		}
	}

	importDataBody.ApiFolderId = strconv.FormatInt(apiFolderId, 10)

	_, err = dghttp.Client11.DoPostJson(ctx, importDataUrl, importDataBody, headers)
	if err != nil {
		panic(err)
	}
}