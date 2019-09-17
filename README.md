# Goku Plugin：Params Transformer

| 插件名称  | 文件名.so |  插件类型  | 错误处理方式 | 作用范围 |  优先级  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |
| 参数映射  | goku-params_transformer | 访问策略 | 继续后续操作 | 转发前  | 801 |

实现表单或json参数的映射，访问API的 **参数A** 绑定到目标API的 **参数B**，映射位置包括header、body、query。

注意事项：
* 若访问API的参数名是user，目标API的参数名是username，此时需开启参数映射插件；若均为username，则无需开启此插件。
* json仅支持 **一级** 映射。
* 若参数类型为表单时，映射插件支持同名参数的使用。
* 使用该插件时请保证Content-Type为 application/x-www-form-urlencoded、 multipart/form-data 或 application/json。

# 目录
- [安装教程](#安装教程 "安装教程")
- [使用教程](#使用教程 "使用教程")
- [更新日志](#更新日志 "更新日志")

# 安装教程
前往 Goku API Gateway 官方网站查看：[插件安装教程](url "https://help.eolinker.com/#/tutorial/?groupID=c-341&productID=19")

# 使用教程

#### 配置页面

进入控制台 >> 策略管理 >> 某策略 >> API插件 >> 参数映射插件：

![](http://data.eolinker.com/course/MciueHY274fe71f0b50c5092e3774aeeccc1a1f29ca9a32)

#### 配置参数

| 参数名 | 说明   | 
| ------------ | ------------ |  
|  paramName | 待映射参数名称 | 
| paramPosition  | 待映射参数所在位置[body/header/query] |
| proxyParamName  | 目标参数名称 |   
| proxyParamPosition  | 目标参数所在位置 |  
| required  | 是否必含，如为true，该参数不存在时会报错 | 
| removeAfterTransformed  | 映射后删除原参数[true/false] | 
| paramConflictSolution  |  参数冲突时的处理方式 [origin/convert/error] |

参数冲突说明：
参数映射插件配置了参数A转换成参数B，但是直接请求时既传了A，又传了B，此时为参数出现冲突，参数B实际上会接收两个参数值。
* convert：参数出现冲突时，取映射后的参数，即A
* origin：参数出现冲突时，取映射前的参数，即B
* error：请求时报错，"param_name"has a conflict.

若paramConflictSolution为空，视为使用默认值convert。

#### 配置示例

```
{
    "params": [
        {
            "paramName": "userName", 
            "paramPosition": "body", 
            "proxyParamName": "loginCall",
            "proxyParamPosition": "query",
             "paramConflictSolution":"convert",
            "required": true 
        },
        {
            "paramName": "password",
            "paramPosition": "body", 
            "proxyParamName": "loginPassword",
            "proxyParamPosition": "query",
            "paramConflictSolution":"convert",
            "required": true
        }
    ],
    "removeAfterTransformed": true
}
```

#### 请求示例

#### 1. required为true[该参数为必填]

* 映射前：

```
curl http://goku:6689/{proxy path} \
    -d 'userName=<user_name>'
```
* 根据上面的配置示例经过映射后:

```
curl http://goku:6689/{proxy path}?loginCall=<user_name> \
    -d 'userName=<user_name>'
```

#### 2. Restful格式

如果请求参数是Restful格式，默认直接映射到转发地址的对应参数，此时 **不需要** 使用参数映射插件。

**接口设置示例**

 * 网关请求路径：

    ```
    /test/:user_name/:pwd
    ```
  
 * 映射路径:

    ```
    /login/:user_name/:pwd
    ```
    
**请求示例**：

 * 映射前：

    ```
    /test/user_1/hashed_pwd
    ```

 * 映射后：

    ```
    /login/user_1/hashed_pwd
    ```