package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	goku_plugin "github.com/eolinker/goku-plugin"
)

const pluginName = "goku-params_transformer"

const (
	FormParamType string = "application/x-www-form-urlencoded"
	JsonType      string = "application/json"
	MultipartType string = "multipart/form-data"
)

var (
	paramConvert string = "convert"
	paramError   string = "error"
	paramOrigin  string = "origin"
)

var builder = new(gokuParamsTransformerPluginFactory)

func Builder() goku_plugin.PluginFactory {
	return builder
}

type gokuParamsTransformerPluginFactory struct {
}

func (f *gokuParamsTransformerPluginFactory) Create(config string, clusterName string, updateTag string, strategyId string, apiId int) (*goku_plugin.PluginObj, error) {

	if config == "" {
		return nil, errors.New("config is empty")
	}
	var conf paramsTransformerconf

	if err := json.Unmarshal([]byte(config), &conf); err != nil {
		return nil, fmt.Errorf("[params_transformer] error occurred while parsing plugin config:%s", err.Error())
	}

	p := &gokuParamsTransformer{
		conf: &conf,
	}
	return &goku_plugin.PluginObj{
		BeforeMatch: nil,
		Access:      p,
		Proxy:       nil,
	}, nil
}

type paramsTransformerparam struct {
	ParamName             string `json:"paramName"`
	ParamPosition         string `json:"paramPosition"`
	ProxyParamName        string `json:"proxyParamName"`
	ProxyParamPosition    string `json:"proxyParamPosition"`
	Required              bool   `json:"required"`
	ParamConflictSolution string `json:"paramConflictSolution"`
}

type paramsTransformerconf struct {
	Params                 []paramsTransformerparam `json:"params"`
	RemoveAfterTransformed bool                     `json:"removeAfterTransformed"`
}

type gokuParamsTransformer struct {
	conf *paramsTransformerconf
}

func writeTOProxy(position string, name string, value string, ctx goku_plugin.ContextAccess) {
	switch position {
	case "header":
		{
			proxyParamName := ConvertHearderKey(name)
			ctx.SetHeader(proxyParamName, value)
		}
	case "query":
		{
			ctx.Proxy().Querys().Set(name, value)
		}
	}
}

func parseBodyParams(ctx goku_plugin.ContextAccess, body []byte, contentType string) (map[string]interface{}, map[string][]string, map[string]*goku_plugin.FileHeader, error) {
	formParams := make(map[string][]string)
	bodyParams := make(map[string]interface{})
	files := make(map[string]*goku_plugin.FileHeader)
	var err error
	if strings.Contains(contentType, FormParamType) {
		formParams, err = ctx.Request().BodyForm()
		if err != nil {
			return bodyParams, formParams, files, err
		}
	} else if strings.Contains(contentType, JsonType) {

		if string(body) != "" {
			err = json.Unmarshal(body, &bodyParams)
			if err != nil {
				return bodyParams, formParams, files, err
			}
		}
	} else if strings.Contains(contentType, MultipartType) {
		formParams, err := ctx.Request().BodyForm()
		if err != nil {
			return bodyParams, formParams, files, err
		}
		files, err := ctx.Request().Files()
		if err != nil {
			return bodyParams, formParams, files, err
		}
	}

	return bodyParams, formParams, files, nil
}

func getHeaderValue(headers map[string][]string, param *paramsTransformerparam, ctx goku_plugin.ContextAccess) (error, string, []string) {
	paramName := ConvertHearderKey(param.ParamName)
	paramValue := []string{}
	if _, ok := headers[paramName]; !ok {
		errInfo := "[params_transformer] param " + param.ParamName + " required"
		return errors.New(errInfo), errInfo, []string{}
	} else {
		paramValue = headers[paramName]
	}
	return nil, "", paramValue
}

func getQueryValue(queryParams map[string][]string, param *paramsTransformerparam, ctx goku_plugin.ContextAccess) (error, string, []string) {
	value := []string{}
	if _, ok := queryParams[param.ParamName]; !ok {
		errInfo := "[params_transformer] param " + param.ParamName + " required"
		return errors.New(errInfo), errInfo, []string{}
	} else {
		value = queryParams[param.ParamName]
	}
	return nil, "", value
}

func getBodyValue(bodyParams map[string]interface{}, formParams map[string][]string, files map[string]*goku_plugin.FileHeader, param *paramsTransformerparam, contentType string, ctx goku_plugin.ContextAccess) (error, string, interface{}) {
	var value interface{} = nil
	errInfo := "[params_transformer] param " + param.ParamName + " required"
	if strings.Contains(contentType, FormParamType) {
		if _, ok := formParams[param.ParamName]; !ok {
			return errors.New(errInfo), errInfo, ""
		} else {
			value = formParams[param.ParamName]
		}
	} else if strings.Contains(contentType, JsonType) {

		if _, ok := bodyParams[param.ParamName]; !ok {
			return errors.New(errInfo), errInfo, ""
		} else {
			value = bodyParams[param.ParamName]
		}
	} else if strings.Contains(contentType, MultipartType) {
		if _, ok := formParams[param.ParamName]; !ok {
			if _, fileOk := files[param.ParamName]; !fileOk {
				return errors.New(errInfo), errInfo, ""
			} else {
				value = files[param.ParamName]
			}
		} else {
			value = bodyParams[param.ParamName]
		}
	}
	return nil, "", value
}

func getProxyValue(position, proxyPosition, paramName, contentType string, headerValue, queryValue []string, bodyValue interface{}) (error, string, []string, interface{}) {
	value := []string{}
	var bodyContent interface{} = nil
	// errInfo := `"[params_transformer] Illegal "paramProxyPosition" in "` + paramName + `"`
	if position == "header" {
		value = append(value, headerValue...)
	} else if position == "query" {
		value = append(value, queryValue...)
	} else if position == "body" {
		if strings.Contains(contentType, FormParamType) || strings.Contains(contentType, MultipartType) {
			if v, ok := bodyValue.([]string); ok {
				value = append(value, v...)
			} else {
				bodyContent = bodyValue
			}
		} else if strings.Contains(contentType, JsonType) {
			if proxyPosition == "body" {
				bodyContent = bodyValue
			} else {
				v, _ := json.Marshal(bodyValue)
				value = append(value, string(v))
			}
		}
	}
	return nil, "", value, bodyContent
}

// 转发前执行
func (pm *gokuParamsTransformer) Access(ctx goku_plugin.ContextAccess) (bool, error) {
	conf := pm.conf
	if conf == nil {
		return true, nil
	}
	contentType := ctx.Request().ContentType()
	body, _ := ctx.Request().RawBody()
	bodyParams, formParams, files, err := parseBodyParams(ctx, body, contentType)
	if err != nil {
		ctx.SetStatus(400, "400")
		ctx.SetBody([]byte(err.Error()))
		return false, err
	}
	headers := ctx.Headers()
	queryParams := ctx.Request().URL().Query()
	for _, param := range conf.Params {
		headerValue := []string{}
		queryValue := []string{}
		var bodyValue interface{} = nil
		switch param.ParamPosition {
		case "header":
			{
				var err error
				var errInfo string
				err, errInfo, headerValue = getHeaderValue(headers, &param, ctx)
				if err != nil {
					if param.Required {
						ctx.SetStatus(400, "400")
						ctx.SetBody([]byte(errInfo))
						return false, err
					} else {
						continue
					}
				}
				if conf.RemoveAfterTransformed {
					ctx.Proxy().DelHeader(param.ParamName)
				}
			}
		case "query":
			{
				var err error
				var errInfo string
				err, errInfo, queryValue = getQueryValue(queryParams, &param, ctx)
				if err != nil {
					if param.Required {
						ctx.SetStatus(400, "400")
						ctx.SetBody([]byte(errInfo))
						return false, err
					} else {
						continue
					}
				}
				if conf.RemoveAfterTransformed {
					ctx.Proxy().Querys().Del(param.ParamName)
				}
			}
		case "body":
			{
				var err error
				var errInfo string
				err, errInfo, bodyValue = getBodyValue(bodyParams, formParams, files, &param, contentType, ctx)
				if err != nil {
					if param.Required {
						ctx.SetStatus(400, "400")
						ctx.SetBody([]byte(errInfo))
						return false, err
					} else {
						continue
					}
				}
				if conf.RemoveAfterTransformed {
					if strings.Contains(contentType, JsonType) {
						delete(bodyParams, param.ParamName)
					} else if strings.Contains(contentType, FormParamType) {
						delete(formParams, param.ParamName)
					} else if strings.Contains(contentType, MultipartType) {
						delete(files, param.ParamName)
					}
				}
			}
		default:
			{
				errInfo := `"[params_transformer] Illegal "paramPosition" in "` + param.ParamName + `"`
				ctx.SetStatus(400, "400")
				ctx.SetBody([]byte(errInfo))
				return false, errors.New(errInfo)
			}
		}

		switch param.ProxyParamPosition {
		case "header":
			{
				err, errInfo, value, _ := getProxyValue(param.ParamPosition, param.ProxyParamPosition, param.ParamName, contentType, headerValue, queryValue, bodyValue)
				if err != nil {
					ctx.SetStatus(400, "400")
					ctx.SetBody([]byte(errInfo))
					return false, err
				}
				if ctx.Proxy().GetHeader(param.ProxyParamName) != "" {
					ctx.Proxy().AddHeader(param.ProxyParamName, value[0])
				} else {
					ctx.Proxy().SetHeader(param.ProxyParamName, value[0])
				}
			}
		case "query":
			{
				err, errInfo, value, _ := getProxyValue(param.ParamPosition, param.ProxyParamPosition, param.ParamName, contentType, headerValue, queryValue, bodyValue)
				if err != nil {
					ctx.SetStatus(400, "400")
					ctx.SetBody([]byte(errInfo))
					return false, err
				}
				for _, v := range value {
					if ctx.Proxy().Querys().Get(param.ProxyParamName) != "" {
						ctx.Proxy().Querys().Add(param.ProxyParamName, v)
					} else {
						ctx.Proxy().Querys().Set(param.ProxyParamName, v)
					}
				}

			}
		case "body":
			{
				err, errInfo, value, bv := getProxyValue(param.ParamPosition, param.ProxyParamPosition, param.ParamName, contentType, headerValue, queryValue, bodyValue)
				if err != nil {
					ctx.SetStatus(400, "400")
					ctx.SetBody([]byte(errInfo))
					return false, err
				}
				if strings.Contains(contentType, FormParamType) {
					if _, ok := formParams[param.ProxyParamName]; ok {
						formParams[param.ProxyParamName] = append(formParams[param.ProxyParamName], value...)
					} else {
						formParams[param.ProxyParamName] = value
					}
				} else if strings.Contains(contentType, JsonType) {
					bodyParams[param.ProxyParamName] = bv
				} else if strings.Contains(contentType, MultipartType) {
					if len(value) > 0 {
						if _, ok := formParams[param.ProxyParamName]; ok {
							formParams[param.ProxyParamName] = append(formParams[param.ProxyParamName], value...)
						} else {
							formParams[param.ProxyParamName] = value
						}
					} else {
						ctx.Proxy().AddFile(param.ProxyParamName, bv.(*goku_plugin.FileHeader))
						// files[param.ProxyParamName] = bv.(*goku_plugin.FileHeader)
					}
				} else {
					continue
				}
			}
		default:
			{
				errInfo := `"[params_transformer] Illegal "paramPosition" in "` + param.ParamName + `"`
				ctx.SetStatus(400, "400")
				ctx.SetBody([]byte(errInfo))
				return false, errors.New(errInfo)
			}
		}
	}
	if strings.Contains(contentType, FormParamType) {
		ctx.Proxy().SetForm(formParams)
	} else if strings.Contains(contentType, JsonType) {
		bodyByte, _ := json.Marshal(bodyParams)
		ctx.Proxy().SetRaw(contentType, bodyByte)
	}

	return true, nil
}
