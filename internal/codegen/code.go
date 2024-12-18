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
		"lower": func(s string) string { return strings.ToLower(s) },
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
	if strings.HasSuffix(out, "\\") {
		out = out[:len(out)-1]
	}

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
{{- range $i, $formData := .Request.Body.FormData }}
	{{- if $formData.Enable }}
    "{{ .Key }}": "{{ .Value }}"{{ if not (last $i $.Request.Body.FormData) }},{{ end }}
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
	if requestSpec.Method == "" {
		requestSpec.Method = domain.RequestMethodGET
	}

	if requestSpec.Method == domain.RequestMethodHEAD {
		requestSpec.Method = "--head"
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
{{- range $i, $field := .Request.Body.FormData }}
    -F "{{ $field.Key }}={{ $field.Value }}"{{ if not (last $i $.Request.Body.FormData) }} \{{ end }}
{{- end }}
{{- end }}
`

	return svc.generate(curlTemplate, requestSpec)
}

func (svc *Service) GenerateAxiosCommand(requestSpec *domain.HTTPRequestSpec) (string, error) {
	const axiosTemplate = `const axios = require('axios');
axios({
    method: '{{ .Method }}',
    url: '{{ .URL }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
        "{{ $header.Key }}": "{{ $header.Value }}"{{ if not (last $i $.Request.Headers) }},{{ end }}
        {{- end }}
    },
    {{- end }}
    {{- if eq .Request.Body.Type "json" }}
    data: {{ .Request.Body.Data }},
    {{- else if eq .Request.Body.Type "text" }}
    data: "{{ .Request.Body.Data }}",
    {{- else if eq .Request.Body.Type "formData" }}
    data: new FormData(),
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
	const fetchTemplate = `fetch('{{ .URL }}', {
    method: '{{ .Method }}',
    {{- if .Request.Headers }}
    headers: {
        {{- range $i, $header := .Request.Headers }}
        "{{ $header.Key }}": "{{ $header.Value }}"{{ if not (last $i $.Request.Headers) }},{{ end }}
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
	const kotlinTemplate = `import okhttp3.*
import java.io.IOException

val client = OkHttpClient()

val request = Request.Builder()
    .url("{{ .URL }}")
    .method("{{ .Method }}", {{ if eq .Request.Body.Type "json" }}RequestBody.create(MediaType.parse("application/json"), "{{ .Request.Body.Data }}"){{ else }}null{{ end }})
    {{- range .Request.Headers }}
    .addHeader("{{ .Key }}", "{{ .Value }}")
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
	const javaTemplate = `import okhttp3.*;
import java.io.IOException;

public class ApiRequest {
    public static void main(String[] args) {
        OkHttpClient client = new OkHttpClient();

        Request request = new Request.Builder()
            .url("{{ .URL }}")
            .method("{{ .Method }}", {{ if eq .Request.Body.Type "json" }}RequestBody.create(MediaType.parse("application/json"), "{{ .Request.Body.Data }}"){{ else }}null{{ end }})
            {{- range .Request.Headers }}
            .addHeader("{{ .Key }}", "{{ .Value }}")
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
	const rubyTemplate = `require 'net/http'
require 'uri'
require 'json'

uri = URI.parse("{{ .URL }}")
http = Net::HTTP.new(uri.host, uri.port)
http.use_ssl = uri.scheme == "https"

request = Net::HTTP::{{ .Method | titleize }}.new(uri.request_uri)
{{- range .Request.Headers }}
request["{{ .Key }}"] = "{{ .Value }}"
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
	const dotNetTemplate = `using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

public class Program {
    private static readonly HttpClient client = new HttpClient();

    public static async Task Main(string[] args) {
        var url = "{{ .URL }}";
        var request = new HttpRequestMessage(HttpMethod.{{ .Method | titleize }}, url);
        
        {{- range .Request.Headers }}
        request.Headers.Add("{{ .Key }}", "{{ .Value }}");
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
