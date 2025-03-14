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

	return r
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
			default:
				return false
			}
		},
		"lower": strings.ToLower,
		"titleize": func(s string) string {
			return cases.Title(language.English).String(strings.ToLower(s))
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
	spec := requestSpec.Clone()
	if spec.Method == domain.RequestMethodHEAD {
		spec.Method = "--head"
	}

	// Define a Go template to generate the `curl` command
	const curlTemplate = `curl -X {{ .Method }} "{{ .URL }}"{{ if .Request.Headers }} \
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

	return svc.generate(curlTemplate, spec)
}

func (svc *Service) GenerateGoRequest(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const goTemplate = `package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	  url := "{{ .URL }}"

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
	  {{- end }}
	  client := &http.Client{}

	  req, err := http.NewRequest("{{ .Method }}", url, payload)
	  if err != nil {
	  	  fmt.Println(err)
	  	  return
      }
{{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
	  req.Header.Add("{{ $header.Key }}", "{{ $header.Value }}")
	{{- end }}
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
	const axiosTemplate = `// This template is not completed yet, please use it as inspiration. 
const axios = require('axios');
axios({
    method: '{{ .Method }}',
    url: '{{ .URL }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        "{{ $header.Key }}": "{{ $header.Value }}"{{ if not (last $i $.Request.Headers) }},{{ end }}
		{{- end }}
        {{- end }}
    },
    {{- end }}
    {{- if eq .Request.Body.Type "json" }}
    data: {{ .Request.Body.Data }},
    {{- else if eq .Request.Body.Type "text" }}
    data: "{{ .Request.Body.Data }}",
    {{- else if eq .Request.Body.Type "formData" }}
    data: new FormData(),
	{{- range $i, $field := .Request.Body.FormData.Fields }}
		{{- if $field.Enable }}
			{{- if eq $field.Type "file" }}
				{{- if eq (len $field.Files) 1 }}
	data.append("{{ $field.Key }}", new Blob([new Uint8Array(Buffer.from(fs.readFileSync("{{ index $field.Files 0 }}")))], { type: 'application/octet-stream' }))
				{{- else }}
					{{- range $j, $file := $field.Files }}
	data.append("{{ $field.Key }}", new Blob([new Uint8Array(Buffer.from(fs.readFileSync("{{ $file }}")))], { type: 'application/octet-stream' }){{ if not (last $j $field.Files) }},{{ end }}
					{{- end }}
				{{- end }}
			{{- else }}
	data.append("{{ $field.Key }}", "{{ $field.Value }}")
			{{- end }}
        {{- end }}
    {{- end }}
    {{- end }}
}).then(response => {
    console.log(response.data);
}).catch(error => {
    console.error(error);
});
`
	return svc.generate(axiosTemplate, requestSpec)
}

func (svc *Service) GenerateFetchCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const fetchTemplate = `// This template is not completed yet, please use it as inspiration. 
fetch('{{ .URL }}', {
    method: '{{ .Method }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        "{{ $header.Key }}": "{{ $header.Value }}"{{ if not (last $i $.Request.Headers) }},{{ end }}
		{{- end }}
        {{- end }}
    },
    {{- end }}
    {{- if eq .Request.Body.Type "json" }}
    body: JSON.stringify({{ .Request.Body.Data }}),
    {{- else if eq .Request.Body.Type "text" }}
    body: "{{ .Request.Body.Data }}",
    {{- end }}
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error(error));
`
	return svc.generate(fetchTemplate, requestSpec)
}

func (svc *Service) GenerateKotlinOkHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const kotlinTemplate = `// This template is not completed yet, please use it as inspiration. 
import okhttp3.*
import java.io.IOException

val client = OkHttpClient()

val request = Request.Builder()
    .url("{{ .URL }}")
    .method("{{ .Method }}", {{ if eq .Request.Body.Type "json" }}RequestBody.create(MediaType.parse("application/json"), "{{ .Request.Body.Data }}"){{ else }}null{{ end }})
    {{- range $i, $header := .Request.Headers }}
	{{- if $header.Enable }}
    .addHeader("{{ .Key }}", "{{ .Value }}")
    {{- end }}
    {{- end }}
    .build()

client.newCall(request).enqueue(object : Callback {
    override fun onFailure(call: Call, e: IOException) {
        e.printStackTrace()
    }

    override fun onResponse(call: Call, response: Response) {
        response.body()?.string()?.let { println(it) }
    }
})
`
	return svc.generate(kotlinTemplate, requestSpec)
}

func (svc *Service) GenerateJavaOkHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const javaTemplate = `// This template is not completed yet, please use it as inspiration. 
import okhttp3.*;
import java.io.IOException;

public class ApiRequest {
    public static void main(String[] args) {
        OkHttpClient client = new OkHttpClient();

        Request request = new Request.Builder()
            .url("{{ .URL }}")
            .method("{{ .Method }}", {{ if eq .Request.Body.Type "json" }}RequestBody.create(MediaType.parse("application/json"), "{{ .Request.Body.Data }}"){{ else }}null{{ end }})
			{{- range $i, $header := .Request.Headers }}
			{{- if $header.Enable }}
            .addHeader("{{ .Key }}", "{{ .Value }}")
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
                System.out.println(response.body().string());
            }
        });
    }
}
`
	return svc.generate(javaTemplate, requestSpec)
}

func (svc *Service) GenerateRubyNetHttpCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const rubyTemplate = `# This template is not completed yet, please use it as inspiration. 
require 'net/http'
require 'uri'
require 'json'

uri = URI.parse("{{ .URL }}")
http = Net::HTTP.new(uri.host, uri.port)
http.use_ssl = uri.scheme == "https"

request = Net::HTTP::{{ .Method | titleize }}.new(uri.request_uri)
{{- range $i, $header := .Request.Headers }}
{{- if $header.Enable }}
request["{{ .Key }}"] = "{{ .Value }}"
{{- end }}
{{- end }}

{{- if eq .Request.Body.Type "json" }}
request.body = {{ .Request.Body.Data }}
{{- else if eq .Request.Body.Type "text" }}
request.body = "{{ .Request.Body.Data }}"
{{- end }}

response = http.request(request)
puts "Response code: \#{response.code}"
puts "Response body: \#{response.body}"
`

	return svc.generate(rubyTemplate, requestSpec)
}

func (svc *Service) GenerateDotNetHttpClientCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const dotNetTemplate = `// This template is not completed yet, please use it as inspiration. 
using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

public class Program {
    private static readonly HttpClient client = new HttpClient();

    public static async Task Main(string[] args) {
        var url = "{{ .URL }}";
        var request = new HttpRequestMessage(HttpMethod.{{ .Method | titleize }}, url);
        
        {{- range $i, $header := .Request.Headers }}
		{{- if $header.Enable }}
        request.Headers.Add("{{ .Key }}", "{{ .Value }}");
        {{- end }}
        {{- end }}
        
        {{- if eq .Request.Body.Type "json" }}
        request.Content = new StringContent("{{ .Request.Body.Data }}", Encoding.UTF8, "application/json");
        {{- else if eq .Request.Body.Type "text" }}
        request.Content = new StringContent("{{ .Request.Body.Data }}", Encoding.UTF8, "text/plain");
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
