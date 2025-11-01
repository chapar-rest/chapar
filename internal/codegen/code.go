package codegen

import (
	"bytes"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/variables"
)

var DefaultService = New()

type Service struct {
	currentEnvironment *domain.Environment
}

func New() *Service {
	return &Service{}
}

func (svc *Service) OnActiveEnvironmentChange(env *domain.Environment) {
	svc.currentEnvironment = env
}

func (svc *Service) applyVariables(req *domain.HTTPRequestSpec) *domain.HTTPRequestSpec {
	vars := variables.GetVariables()
	r := req.Clone()
	r.RenderParams()
	variables.ApplyToHTTPRequest(vars, r)

	if r.Method == "" {
		r.Method = domain.RequestMethodGET
	}

	if svc.currentEnvironment != nil {
		variables.ApplyToEnv(vars, &svc.currentEnvironment.Spec)
		svc.currentEnvironment.ApplyToHTTPRequest(r)
	}

	// Apply authentication to headers for code generation
	svc.applyAuthToHeaders(r)

	return r
}

func (svc *Service) applyAuthToHeaders(req *domain.HTTPRequestSpec) {
	if req.Request.Auth == (domain.Auth{}) {
		return
	}

	switch req.Request.Auth.Type {
	case domain.AuthTypeToken:
		if req.Request.Auth.TokenAuth != nil && req.Request.Auth.TokenAuth.Token != "" {
			// Check if Authorization header already exists
			hasAuthHeader := false
			for i, h := range req.Request.Headers {
				if h.Key == "Authorization" {
					req.Request.Headers[i].Value = "Bearer " + req.Request.Auth.TokenAuth.Token
					req.Request.Headers[i].Enable = true
					hasAuthHeader = true
					break
				}
			}
			if !hasAuthHeader {
				req.Request.Headers = append(req.Request.Headers, domain.KeyValue{
					Key:    "Authorization",
					Value:  "Bearer " + req.Request.Auth.TokenAuth.Token,
					Enable: true,
				})
			}
		}
	case domain.AuthTypeBasic:
		if req.Request.Auth.BasicAuth != nil &&
			req.Request.Auth.BasicAuth.Username != "" &&
			req.Request.Auth.BasicAuth.Password != "" {
			// For basic auth, we'll add a comment in templates since the encoding varies by language
			// Add as a custom header with the credentials
			basicAuthValue := req.Request.Auth.BasicAuth.Username + ":" + req.Request.Auth.BasicAuth.Password
			hasAuthHeader := false
			for i, h := range req.Request.Headers {
				if h.Key == "Authorization" {
					req.Request.Headers[i].Value = "Basic " + basicAuthValue
					req.Request.Headers[i].Enable = true
					hasAuthHeader = true
					break
				}
			}
			if !hasAuthHeader {
				req.Request.Headers = append(req.Request.Headers, domain.KeyValue{
					Key:    "Authorization",
					Value:  "Basic " + basicAuthValue,
					Enable: true,
				})
			}
		}
	case domain.AuthTypeAPIKey:
		if req.Request.Auth.APIKeyAuth != nil &&
			req.Request.Auth.APIKeyAuth.Key != "" &&
			req.Request.Auth.APIKeyAuth.Value != "" {
			// Add API key as custom header
			hasHeader := false
			for i, h := range req.Request.Headers {
				if h.Key == req.Request.Auth.APIKeyAuth.Key {
					req.Request.Headers[i].Value = req.Request.Auth.APIKeyAuth.Value
					req.Request.Headers[i].Enable = true
					hasHeader = true
					break
				}
			}
			if !hasHeader {
				req.Request.Headers = append(req.Request.Headers, domain.KeyValue{
					Key:    req.Request.Auth.APIKeyAuth.Key,
					Value:  req.Request.Auth.APIKeyAuth.Value,
					Enable: true,
				})
			}
		}
	}
}

func (svc *Service) generate(codeTmpl string, requestSpec *domain.HTTPRequestSpec) (string, error) {
	req := svc.applyVariables(requestSpec)

	// Parse and execute the template
	tmpl, err := template.New("code-template").Funcs(template.FuncMap{
		"last": func(i int, list interface{}) bool {
			switch v := list.(type) {
			case []domain.KeyValue:
				return i == len(v)-1
			case []domain.FormField:
				return i == len(v)-1
			case []string:
				return i == len(v)-1
			default:
				return false
			}
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"titleize": func(s string) string {
			return cases.Title(language.English).String(strings.ToLower(s))
		},
		"escape": func(s string) string {
			s = strings.ReplaceAll(s, "\\", "\\\\")
			s = strings.ReplaceAll(s, "\"", "\\\"")
			s = strings.ReplaceAll(s, "\n", "\\n")
			s = strings.ReplaceAll(s, "\r", "\\r")
			s = strings.ReplaceAll(s, "\t", "\\t")
			return s
		},
		"join": func(sep string, list []string) string {
			return strings.Join(list, sep)
		},
	}).Parse(codeTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, req); err != nil {
		return "", err
	}
	out := buf.String()

	// trim the last backslash
	out = strings.TrimSpace(out)
	out = strings.TrimSuffix(out, "\\")
	out += "\n"

	return out, nil
}

func (svc *Service) GeneratePythonRequest(requestSpec *domain.HTTPRequestSpec) (string, error) {
	// Define a Go template to generate the Python `requests` code
	const pythonTemplate = `import requests
{{- if eq .Request.Body.Type "json" }}
import json
{{- end }}

url = "{{ .URL }}"

headers = {
{{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
     "{{ .Key }}": "{{ .Value }}"{{ if not (last $i $.Request.Headers) }},{{ end }}
	{{- end }}
{{- end }}
}

{{- if eq .Request.Body.Type "json" }}
json_data = json.dumps({{ .Request.Body.Data }})
{{- else if eq .Request.Body.Type "text" }}
data = '''{{ .Request.Body.Data }}'''
{{- else if eq .Request.Body.Type "formData" }}
files = {
{{- range $i, $formData := .Request.Body.FormData.Fields }}
	{{- if $formData.Enable }}
	{{- if eq $formData.Type "file" }}
		{{ if eq (len $formData.Files) 1 }}
	"{{ $formData.Key }}": open("{{ index $formData.Files 0 }}", "rb"){{ if not (last $i $.Request.Body.FormData.Fields) }},{{ end }}
		{{- else }}
	"{{ $formData.Key }}": [open(file, "rb") for file in [{{- range $j, $file := $formData.Files }}"{{ $file }}"{{ if not (last $j $formData.Files) }},{{ end }}{{- end }}]]{{ if not (last $i $.Request.Body.FormData.Fields) }},{{ end }}
		{{- end }}
	{{- else }}
	"{{ $formData.Key }}": "{{ $formData.Value }}"{{ if not (last $i $.Request.Body.FormData.Fields) }},{{ end }}
	{{- end }}
	{{- end }}
{{- end }}
}
{{- else if eq .Request.Body.Type "urlEncoded" }}
data = {
{{- range $i, $field := .Request.Body.URLEncoded }}
	{{- if $field.Enable }}
	"{{ $field.Key }}": "{{ $field.Value }}"{{ if not (last $i $.Request.Body.URLEncoded) }},{{ end }}
	{{- end }}
{{- end }}
}
{{- else if eq .Request.Body.Type "binary" }}
with open("{{ .Request.Body.BinaryFilePath }}", "rb") as f:
    data = f.read()
{{- else }}
data = None
{{- end }}

response = requests.{{ .Method | lower }}(
    url, headers=headers,
{{- if eq .Request.Body.Type "json" }}
    data=json_data
{{- else if eq .Request.Body.Type "text" }}
    data=data
{{- else if eq .Request.Body.Type "formData" }}
    files=files
{{- else if eq .Request.Body.Type "urlEncoded" }}
    data=data
{{- else if eq .Request.Body.Type "binary" }}
    data=data
{{- else }}
    data=data
{{- end }}
)

print(response.status_code)
print(response.text)
`

	return svc.generate(pythonTemplate, requestSpec)
}

func (svc *Service) GenerateCurlCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	// Define a Go template to generate the `curl` command
	const curlTemplate = `{{- if eq .Method "HEAD" }}curl --head "{{ .URL }}"{{- else }}curl -X {{ .Method }} "{{ .URL }}"{{- end }}{{ if .Request.Headers }} \
{{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
    -H "{{ $header.Key }}: {{ $header.Value }}" \
	{{- end }}
{{- end }}
{{- end }}
{{- if eq .Request.Body.Type "json" }}
    -d '{{ .Request.Body.Data }}'
{{- else if eq .Request.Body.Type "text" }}
    --data '{{ .Request.Body.Data }}'
{{- else if eq .Request.Body.Type "formData" }}
	{{- range $i, $field := .Request.Body.FormData.Fields }}
		{{- if $field.Enable }}
    		{{- if eq $field.Type "file" }}
				{{- if eq (len $field.Files) 1 }}
	-F "{{ $field.Key }}=@{{ index $field.Files 0 }}"
				{{- else }}
				{{- range $j, $file := $field.Files }}	
	-F "{{ $field.Key }}[]=@{{ $file }}"{{ if not (last $j $field.Files) }} \{{ end }}
				{{- end }}
				{{- end }}
			{{- else }}
	-F "{{ $field.Key }}={{ $field.Value }}" {{- if not (last $i $.Request.Body.FormData.Fields) }} \{{- end }}
			{{- end }} 
		{{- end }}
    {{- end }}
{{- else if eq .Request.Body.Type "binary" }}
	--data-binary "@{{ .Request.Body.BinaryFilePath }}"
{{- else if eq .Request.Body.Type "urlEncoded" }}
	{{- range $i, $field := .Request.Body.URLEncoded }}
		{{- if $field.Enable }}
	-d "{{ $field.Key }}={{ $field.Value }}"{{ if not (last $i $.Request.Body.URLEncoded) }} \{{ end }}
    	{{- end }}
	{{- end }}
{{- end }}
`

	return svc.generate(curlTemplate, requestSpec)
}

func (svc *Service) GenerateGoRequest(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const goTemplate = `package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	  requestURL := "{{ .URL }}"

      {{- if eq .Request.Body.Type "json" }}
	  payload := strings.NewReader(` + "`{{ .Request.Body.Data }}`" + `)
	  {{- else if eq .Request.Body.Type "text" }}
	  payload := strings.NewReader("{{ .Request.Body.Data }}")
	  {{- else if eq .Request.Body.Type "formData" }}
	  payload := &bytes.Buffer{}
	  writer := multipart.NewWriter(payload)
	  {{- range $i, $field := .Request.Body.FormData.Fields }}
	  	{{- if $field.Enable }}
			{{ if eq $field.Type "file" }}

	  			{{- if eq (len $field.Files) 1 }}

	  file, err := os.Open("{{ index $field.Files 0 }}")
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
	  }
	  defer file.Close()

	  part, err := writer.CreateFormFile("{{ $field.Key }}", filepath.Base("{{ index $field.Files 0 }}"))
	  if err != nil {
	  	  fmt.Println(err)
	  	  return	
	  }

	  if _, err := io.Copy(part, file); err != nil {
	  	  fmt.Println(err)
	  	  return
	  }
				{{- else }}
	  for _, filePath := range []string{ {{- range $j, $file := $field.Files }}"{{ $file }}"{{ if not (last $j $field.Files) }},{{ end }}{{- end }} } {
		  file, err := os.Open(filePath)
		  if err != nil {
			  fmt.Println(err)
			  return
		  }
		  defer file.Close()

		  part, err := writer.CreateFormFile("{{ $field.Key }}", filepath.Base(filePath))
		  if err != nil {
			  fmt.Println(err)
			  return
		 }

		 if _, err := io.Copy(part, file); err != nil {
		   fmt.Println(err)
		   return
		 }
	  }
				{{- end }}
			{{ else }}
	  _ = writer.WriteField("{{ $field.Key }}", "{{ $field.Value }}")
	  		{{- end }}
	  	{{- end }}
	  {{- end }}
	  if err := writer.Close(); err != nil {
	  	  fmt.Println(err)
	  	  return
	  }
	  {{- else if eq .Request.Body.Type "urlEncoded" }}
	  data := url.Values{}
	  {{- range $i, $field := .Request.Body.URLEncoded }}
	  	{{- if $field.Enable }}
	  data.Set("{{ $field.Key }}", "{{ $field.Value }}")
	  	{{- end }}
	  {{- end }}
	  payload := strings.NewReader(data.Encode())
	  {{- else if eq .Request.Body.Type "binary" }}
	  file, err := os.Open("{{ .Request.Body.BinaryFilePath }}")
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
	  }
	  defer file.Close()
	  payload := file
	  {{- else }}
	  var payload io.Reader = nil
	  {{- end }}
	  client := &http.Client{}

	  req, err := http.NewRequest("{{ .Method }}", requestURL, payload)
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
      }
{{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
	  req.Header.Add("{{ $header.Key }}", "{{ $header.Value }}")
	{{- end }}
{{- end }}
{{- if eq .Request.Body.Type "formData" }}
	  req.Header.Set("Content-Type", writer.FormDataContentType())
{{- else if eq .Request.Body.Type "urlEncoded" }}
	  req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
{{- end }}

	  res, err := client.Do(req)
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
      }
	  defer res.Body.Close()
		
	  body, err := io.ReadAll(res.Body)
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
	  }
	  fmt.Println(string(body))
}`

	return svc.generate(goTemplate, requestSpec)
}

func (svc *Service) GenerateAxiosCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const axiosTemplate = `const axios = require('axios');
{{- if eq .Request.Body.Type "formData" }}
const FormData = require('form-data');
const fs = require('fs');
{{- else if eq .Request.Body.Type "binary" }}
const fs = require('fs');
{{- end }}

{{- if eq .Request.Body.Type "formData" }}
const formData = new FormData();
{{- range $i, $field := .Request.Body.FormData.Fields }}
	{{- if $field.Enable }}
		{{- if eq $field.Type "file" }}
			{{- range $j, $file := $field.Files }}
formData.append('{{ $field.Key }}', fs.createReadStream('{{ $file }}'));
			{{- end }}
		{{- else }}
formData.append('{{ $field.Key }}', '{{ $field.Value }}');
		{{- end }}
	{{- end }}
{{- end }}
{{- end }}

axios({
    method: '{{ .Method }}',
    url: '{{ .URL }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        '{{ $header.Key }}': '{{ $header.Value }}'{{ if not (last $i $.Request.Headers) }},{{ end }}
		{{- end }}
        {{- end }}
		{{- if eq .Request.Body.Type "formData" }}
        ,...formData.getHeaders()
		{{- end }}
    },
    {{- end }}
    {{- if eq .Request.Body.Type "json" }}
    data: {{ .Request.Body.Data }},
    {{- else if eq .Request.Body.Type "text" }}
    data: '{{ .Request.Body.Data }}',
    {{- else if eq .Request.Body.Type "formData" }}
    data: formData,
    {{- else if eq .Request.Body.Type "urlEncoded" }}
    data: new URLSearchParams({
		{{- range $i, $field := .Request.Body.URLEncoded }}
			{{- if $field.Enable }}
        '{{ $field.Key }}': '{{ $field.Value }}'{{ if not (last $i $.Request.Body.URLEncoded) }},{{ end }}
			{{- end }}
		{{- end }}
    }),
    {{- else if eq .Request.Body.Type "binary" }}
    data: fs.readFileSync('{{ .Request.Body.BinaryFilePath }}'),
    {{- end }}
}).then(response => {
    console.log(response.status);
    console.log(response.data);
}).catch(error => {
    console.error(error.message);
});
`
	return svc.generate(axiosTemplate, requestSpec)
}

func (svc *Service) GenerateFetchCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const fetchTemplate = `{{- if eq .Request.Body.Type "formData" }}
const FormData = require('form-data');
const fs = require('fs');

const formData = new FormData();
{{- range $i, $field := .Request.Body.FormData.Fields }}
	{{- if $field.Enable }}
		{{- if eq $field.Type "file" }}
			{{- range $j, $file := $field.Files }}
formData.append('{{ $field.Key }}', fs.createReadStream('{{ $file }}'));
			{{- end }}
		{{- else }}
formData.append('{{ $field.Key }}', '{{ $field.Value }}');
		{{- end }}
	{{- end }}
{{- end }}

{{- else if eq .Request.Body.Type "binary" }}
const fs = require('fs');
const binaryData = fs.readFileSync('{{ .Request.Body.BinaryFilePath }}');

{{- end }}
fetch('{{ .URL }}', {
    method: '{{ .Method }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        '{{ $header.Key }}': '{{ $header.Value }}'{{ if not (last $i $.Request.Headers) }},{{ end }}
		{{- end }}
        {{- end }}
		{{- if eq .Request.Body.Type "urlEncoded" }}
        ,'Content-Type': 'application/x-www-form-urlencoded'
		{{- end }}
    },
    {{- end }}
    {{- if eq .Request.Body.Type "json" }}
    body: JSON.stringify({{ .Request.Body.Data }}),
    {{- else if eq .Request.Body.Type "text" }}
    body: '{{ .Request.Body.Data }}',
    {{- else if eq .Request.Body.Type "formData" }}
    body: formData,
    {{- else if eq .Request.Body.Type "urlEncoded" }}
    body: new URLSearchParams({
		{{- range $i, $field := .Request.Body.URLEncoded }}
			{{- if $field.Enable }}
        '{{ $field.Key }}': '{{ $field.Value }}'{{ if not (last $i $.Request.Body.URLEncoded) }},{{ end }}
			{{- end }}
		{{- end }}
    }).toString(),
    {{- else if eq .Request.Body.Type "binary" }}
    body: binaryData,
    {{- end }}
})
.then(response => {
    console.log('Status:', response.status);
    return response.text();
})
.then(data => console.log(data))
.catch(error => console.error(error));
`
	return svc.generate(fetchTemplate, requestSpec)
}

func (svc *Service) GenerateKotlinOkHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const kotlinTemplate = `import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File
import java.io.IOException

val client = OkHttpClient()

{{- if eq .Request.Body.Type "json" }}
val requestBody = """{{ .Request.Body.Data }}""".toRequestBody("application/json".toMediaType())
{{- else if eq .Request.Body.Type "text" }}
val requestBody = """{{ .Request.Body.Data }}""".toRequestBody("text/plain".toMediaType())
{{- else if eq .Request.Body.Type "formData" }}
val requestBody = MultipartBody.Builder()
    .setType(MultipartBody.FORM)
	{{- range $i, $field := .Request.Body.FormData.Fields }}
		{{- if $field.Enable }}
			{{- if eq $field.Type "file" }}
				{{- range $j, $file := $field.Files }}
    .addFormDataPart("{{ $field.Key }}", File("{{ $file }}").name, File("{{ $file }}").asRequestBody("application/octet-stream".toMediaType()))
				{{- end }}
			{{- else }}
    .addFormDataPart("{{ $field.Key }}", "{{ $field.Value }}")
			{{- end }}
		{{- end }}
	{{- end }}
    .build()
{{- else if eq .Request.Body.Type "urlEncoded" }}
val requestBody = FormBody.Builder()
	{{- range $i, $field := .Request.Body.URLEncoded }}
		{{- if $field.Enable }}
    .add("{{ $field.Key }}", "{{ $field.Value }}")
		{{- end }}
	{{- end }}
    .build()
{{- else if eq .Request.Body.Type "binary" }}
val requestBody = File("{{ .Request.Body.BinaryFilePath }}").asRequestBody("application/octet-stream".toMediaType())
{{- else }}
val requestBody = null
{{- end }}

val request = Request.Builder()
    .url("{{ .URL }}")
    .method("{{ .Method }}", requestBody)
    {{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
    .addHeader("{{ $header.Key }}", "{{ $header.Value }}")
    {{- end }}
    {{- end }}
    .build()

client.newCall(request).enqueue(object : Callback {
    override fun onFailure(call: Call, e: IOException) {
        e.printStackTrace()
    }

    override fun onResponse(call: Call, response: Response) {
        println("Status: ${response.code}")
        response.body?.string()?.let { println(it) }
    }
})
`
	return svc.generate(kotlinTemplate, requestSpec)
}

func (svc *Service) GenerateJavaOkHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const javaTemplate = `import okhttp3.*;
import java.io.File;
import java.io.IOException;

public class ApiRequest {
    public static void main(String[] args) {
        OkHttpClient client = new OkHttpClient();

        {{- if eq .Request.Body.Type "json" }}
        RequestBody requestBody = RequestBody.create(
            """{{ .Request.Body.Data }}""",
            MediaType.parse("application/json")
        );
        {{- else if eq .Request.Body.Type "text" }}
        RequestBody requestBody = RequestBody.create(
            """{{ .Request.Body.Data }}""",
            MediaType.parse("text/plain")
        );
        {{- else if eq .Request.Body.Type "formData" }}
        MultipartBody.Builder formBuilder = new MultipartBody.Builder()
            .setType(MultipartBody.FORM);
		{{- range $i, $field := .Request.Body.FormData.Fields }}
			{{- if $field.Enable }}
				{{- if eq $field.Type "file" }}
					{{- range $j, $file := $field.Files }}
        formBuilder.addFormDataPart("{{ $field.Key }}", 
            new File("{{ $file }}").getName(),
            RequestBody.create(new File("{{ $file }}"), MediaType.parse("application/octet-stream"))
        );
					{{- end }}
				{{- else }}
        formBuilder.addFormDataPart("{{ $field.Key }}", "{{ $field.Value }}");
				{{- end }}
			{{- end }}
		{{- end }}
        RequestBody requestBody = formBuilder.build();
        {{- else if eq .Request.Body.Type "urlEncoded" }}
        FormBody.Builder formBuilder = new FormBody.Builder();
		{{- range $i, $field := .Request.Body.URLEncoded }}
			{{- if $field.Enable }}
        formBuilder.add("{{ $field.Key }}", "{{ $field.Value }}");
			{{- end }}
		{{- end }}
        RequestBody requestBody = formBuilder.build();
        {{- else if eq .Request.Body.Type "binary" }}
        RequestBody requestBody = RequestBody.create(
            new File("{{ .Request.Body.BinaryFilePath }}"),
            MediaType.parse("application/octet-stream")
        );
        {{- else }}
        RequestBody requestBody = null;
        {{- end }}

        Request request = new Request.Builder()
            .url("{{ .URL }}")
            .method("{{ .Method }}", requestBody)
			{{- range $i, $header := .Request.Headers }}
			{{- if $header.Enable }}
            .addHeader("{{ $header.Key }}", "{{ $header.Value }}")
            {{- end }}
            {{- end }}
            .build();

        client.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                e.printStackTrace();
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                System.out.println("Status: " + response.code());
                System.out.println(response.body().string());
            }
        });
    }
}
`
	return svc.generate(javaTemplate, requestSpec)
}

func (svc *Service) GenerateRubyNetHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const rubyTemplate = `require 'net/http'
require 'uri'
{{- if eq .Request.Body.Type "json" }}
require 'json'
{{- else if eq .Request.Body.Type "formData" }}
require 'net/http/post/multipart'
{{- end }}

uri = URI.parse("{{ .URL }}")
http = Net::HTTP.new(uri.host, uri.port)
http.use_ssl = uri.scheme == "https"

{{- if eq .Request.Body.Type "formData" }}
{{- range $i, $field := .Request.Body.FormData.Fields }}
	{{- if $field.Enable }}
		{{- if eq $field.Type "file" }}
			{{- range $j, $file := $field.Files }}
file_{{ $i }}_{{ $j }} = File.open("{{ $file }}")
			{{- end }}
		{{- end }}
	{{- end }}
{{- end }}

request = Net::HTTP::Post::Multipart.new(
  uri.request_uri,
  {
	{{- range $i, $field := .Request.Body.FormData.Fields }}
		{{- if $field.Enable }}
			{{- if eq $field.Type "file" }}
				{{- range $j, $file := $field.Files }}
    "{{ $field.Key }}" => UploadIO.new(file_{{ $i }}_{{ $j }}, "application/octet-stream", "{{ $file }}"),
				{{- end }}
			{{- else }}
    "{{ $field.Key }}" => "{{ $field.Value }}",
			{{- end }}
		{{- end }}
	{{- end }}
  }
)
{{- else }}
request = Net::HTTP::{{ .Method | titleize }}.new(uri.request_uri)
{{- end }}

{{- range $i, $header := .Request.Headers }}
{{- if $header.Enable }}
request["{{ $header.Key }}"] = "{{ $header.Value }}"
{{- end }}
{{- end }}

{{- if ne .Request.Body.Type "formData" }}
{{- if eq .Request.Body.Type "json" }}
request.body = {{ .Request.Body.Data }}
request["Content-Type"] = "application/json"
{{- else if eq .Request.Body.Type "text" }}
request.body = "{{ .Request.Body.Data }}"
request["Content-Type"] = "text/plain"
{{- else if eq .Request.Body.Type "urlEncoded" }}
request.set_form_data({
	{{- range $i, $field := .Request.Body.URLEncoded }}
		{{- if $field.Enable }}
  "{{ $field.Key }}" => "{{ $field.Value }}"{{ if not (last $i $.Request.Body.URLEncoded) }},{{ end }}
		{{- end }}
	{{- end }}
})
{{- else if eq .Request.Body.Type "binary" }}
request.body = File.read("{{ .Request.Body.BinaryFilePath }}")
request["Content-Type"] = "application/octet-stream"
{{- end }}
{{- end }}

response = http.request(request)
puts "Response code: #{response.code}"
puts "Response body: #{response.body}"
`

	return svc.generate(rubyTemplate, requestSpec)
}

func (svc *Service) GenerateDotNetHttpClientCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const dotNetTemplate = `using System;
using System.IO;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;
{{- if eq .Request.Body.Type "urlEncoded" }}
using System.Collections.Generic;
{{- end }}

public class Program {
    private static readonly HttpClient client = new HttpClient();

    public static async Task Main(string[] args) {
        var url = "{{ .URL }}";
        var request = new HttpRequestMessage(HttpMethod.{{ .Method | titleize }}, url);
        
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        request.Headers.Add("{{ $header.Key }}", "{{ $header.Value }}");
        {{- end }}
        {{- end }}
        
        {{- if eq .Request.Body.Type "json" }}
        request.Content = new StringContent("""{{ .Request.Body.Data }}""", Encoding.UTF8, "application/json");
        {{- else if eq .Request.Body.Type "text" }}
        request.Content = new StringContent("""{{ .Request.Body.Data }}""", Encoding.UTF8, "text/plain");
        {{- else if eq .Request.Body.Type "formData" }}
        var formData = new MultipartFormDataContent();
		{{- range $i, $field := .Request.Body.FormData.Fields }}
			{{- if $field.Enable }}
				{{- if eq $field.Type "file" }}
					{{- range $j, $file := $field.Files }}
        var fileStream{{ $i }}_{{ $j }} = File.OpenRead("{{ $file }}");
        formData.Add(new StreamContent(fileStream{{ $i }}_{{ $j }}), "{{ $field.Key }}", Path.GetFileName("{{ $file }}"));
					{{- end }}
				{{- else }}
        formData.Add(new StringContent("{{ $field.Value }}"), "{{ $field.Key }}");
				{{- end }}
			{{- end }}
		{{- end }}
        request.Content = formData;
        {{- else if eq .Request.Body.Type "urlEncoded" }}
        var formData = new FormUrlEncodedContent(new[] {
			{{- range $i, $field := .Request.Body.URLEncoded }}
				{{- if $field.Enable }}
            new KeyValuePair<string, string>("{{ $field.Key }}", "{{ $field.Value }}"){{ if not (last $i $.Request.Body.URLEncoded) }},{{ end }}
				{{- end }}
			{{- end }}
        });
        request.Content = formData;
        {{- else if eq .Request.Body.Type "binary" }}
        var fileBytes = File.ReadAllBytes("{{ .Request.Body.BinaryFilePath }}");
        request.Content = new ByteArrayContent(fileBytes);
        request.Content.Headers.ContentType = new System.Net.Http.Headers.MediaTypeHeaderValue("application/octet-stream");
        {{- end }}

        HttpResponseMessage response = await client.SendAsync(request);
        string responseBody = await response.Content.ReadAsStringAsync();
        Console.WriteLine("Response Code: " + response.StatusCode);
        Console.WriteLine("Response Body: " + responseBody);
    }
}
`

	return svc.generate(dotNetTemplate, requestSpec)
}
