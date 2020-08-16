package ktgen

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/tal-tech/go-zero/tools/goctl/api/spec"
)

const (
	apiBaseTemplate = `package {{.}}

import com.google.gson.Gson
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL

const val SERVER = "http://localhost:8080"

suspend fun apiRequest(
    method: String,
    uri: String,
    body: Any = "",
    onOk: ((String) -> Unit)? = null,
    onFail: ((String) -> Unit)? = null,
    eventually: (() -> Unit)? = null
) = withContext(Dispatchers.IO) {
    val url = URL(SERVER + uri)
    with(url.openConnection() as HttpURLConnection) {
        requestMethod = method
        doInput = true
        if (method == "POST" || method == "PUT") {
            setRequestProperty("Content-Type", "application/json")
            doOutput = true
            val data = when (body) {
                is String -> {
                    body
                }
                else -> {
                    Gson().toJson(body)
                }
            }
            val wr = OutputStreamWriter(outputStream)
            wr.write(data)
            wr.flush()
        }
        if (responseCode >= 400) {
            BufferedReader(InputStreamReader(errorStream)).use {
                val response = it.readText()
                onFail?.invoke(response)
            }
            return@with
        }
        //response
        BufferedReader(InputStreamReader(inputStream)).use {
            val response = it.readText()
            onOk?.invoke(response)
        }
    }
    eventually?.invoke()
}
`
	apiTemplate = `package {{with .Info}}{{.Desc}}{{end}}

import com.google.gson.Gson

object {{with .Info}}{{.Title}}{{end}}{
	{{range .Types}}
	data class {{.Name}}({{$length := (len .Members)}}{{range $i,$item := .Members}}
		val {{with $item}}{{lowCamelCase .Name}}: {{parseType .Type}}{{end}}{{if ne $i (add $length -1)}},{{end}}{{end}}
	){{end}}
	{{with .Service}}
	{{range .Routes}}suspend fun {{pathToFuncName .Path}}({{with .RequestType}}{{if ne .Name ""}}
		req:{{.Name}},{{end}}{{end}}
		onOk: (({{with .ResponseType}}{{.Name}}{{end}}) -> Unit)? = null,
        onFail: ((String) -> Unit)? = null,
        eventually: (() -> Unit)? = null
    ){
        apiRequest("{{upperCase .Method}}","{{.Path}}",{{with .RequestType}}{{if ne .Name ""}}body=req,{{end}}{{end}} onOk = { {{with .ResponseType}}
            onOk?.invoke({{if ne .Name ""}}Gson().fromJson(it,{{.Name}}::class.java){{end}}){{end}}
        }, onFail = onFail, eventually =eventually)
    }
	{{end}}{{end}}
}`
)

func genBase(dir, pkg string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "BaseApi.kt")
	if _, e := os.Stat(path); e == nil {
		fmt.Println("BaseApi.kt already exists, skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New("n").Parse(apiBaseTemplate)
	if e != nil {
		return e
	}
	e = t.Execute(file, pkg)
	if e != nil {
		return e
	}
	return nil
}

func genApi(dir, pkg string, api *spec.ApiSpec) error {
	name := strcase.ToCamel(api.Info.Title + "Api")
	path := filepath.Join(dir, name+".kt")
	api.Info.Title = name
	api.Info.Desc = pkg

	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New("api").Funcs(funcsMap).Parse(apiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}
