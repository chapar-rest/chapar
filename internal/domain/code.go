package domain

import (
	"bytes"
	"strings"
	"text/template"
)

func GeneratePythonRequest(requestSpec HTTPRequestSpec) (string, error) {
	// Define a Go template to generate the Python `requests` code
	const pythonTemplate = `
import requests

url = "{{ .URL }}"

headers = {
{{- range .Request.Headers }}
    "{{ .Key }}": "{{ .Value }}",
{{- end }}
}

params = {
{{- range .Request.QueryParams }}
    "{{ .Key }}": "{{ .Value }}",
{{- end }}
}

{{- if eq .Request.Body.Type "json" }}
json_data = {{ .Request.Body.Data }}
{{- else if eq .Request.Body.Type "text" }}
data = '''{{ .Request.Body.Data }}'''
{{- else if eq .Request.Body.Type "formData" }}
files = {
{{- range .Request.Body.FormData }}
    "{{ .Key }}": "{{ .Value }}",
{{- end }}
}
{{- else }}
data = None
{{- end }}

response = requests.{{ .Method | lower }}(
    url, headers=headers, params=params,
{{- if eq .Request.Body.Type "json" }}
    json=json_data
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

	// Parse and execute the template
	tmpl, err := template.New("pythonRequest").Funcs(template.FuncMap{
		"lower": func(s string) string {
			return string(bytes.ToLower([]byte(s)))
		},
	}).Parse(pythonTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateCurlCommand(requestSpec HTTPRequestSpec) (string, error) {
	// Define a Go template to generate the `curl` command
	const curlTemplate = `
curl -X {{ .Method }} "{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}"{{ if .Request.Headers }} \
{{- range $i, $header := .Request.Headers }}
    -H "{{ $header.Key }}: {{ $header.Value }}"{{ if not (last $i $.Request.Headers) }} \{{ end }}
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

	// Parse and execute the template
	tmpl, err := template.New("curlCommand").Funcs(template.FuncMap{
		"last": func(i int, list interface{}) bool {
			switch v := list.(type) {
			case []KeyValue:
				return i == len(v)-1
			case []FormField:
				return i == len(v)-1
			default:
				return false
			}
		},
	}).Parse(curlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateAxiosCommand(requestSpec HTTPRequestSpec) (string, error) {
	const axiosTemplate = `
const axios = require('axios');

axios({
    method: '{{ .Method }}',
    url: '{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}',
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
	// Same helper function as before to detect the last element
	tmpl, err := template.New("axiosCommand").Funcs(template.FuncMap{
		"last": func(i int, list interface{}) bool {
			switch v := list.(type) {
			case []KeyValue:
				return i == len(v)-1
			case []FormField:
				return i == len(v)-1
			default:
				return false
			}
		},
	}).Parse(axiosTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateFetchCommand(requestSpec HTTPRequestSpec) (string, error) {
	const fetchTemplate = `
fetch('{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}', {
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
	// Parse and execute the template
	tmpl, err := template.New("fetchCommand").Funcs(template.FuncMap{
		"last": func(i int, list interface{}) bool {
			switch v := list.(type) {
			case []KeyValue:
				return i == len(v)-1
			case []FormField:
				return i == len(v)-1
			default:
				return false
			}
		},
	}).Parse(fetchTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateKotlinOkHttpCommand(requestSpec HTTPRequestSpec) (string, error) {
	const kotlinTemplate = `
import okhttp3.*
import java.io.IOException

val client = OkHttpClient()

val request = Request.Builder()
    .url("{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}")
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
	tmpl, err := template.New("kotlinCommand").Parse(kotlinTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateJavaOkHttpCommand(requestSpec HTTPRequestSpec) (string, error) {
	const javaTemplate = `
import okhttp3.*;
import java.io.IOException;

public class ApiRequest {
    public static void main(String[] args) {
        OkHttpClient client = new OkHttpClient();

        Request request = new Request.Builder()
            .url("{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}")
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
	tmpl, err := template.New("javaCommand").Parse(javaTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateRubyNetHttpCommand(requestSpec HTTPRequestSpec) (string, error) {
	const rubyTemplate = `
require 'net/http'
require 'uri'
require 'json'

uri = URI.parse("{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}")
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

	// Parse and execute the template
	tmpl, err := template.New("rubyCommand").Funcs(template.FuncMap{
		"titleize": func(s string) string {
			return strings.Title(strings.ToLower(s))
		},
	}).Parse(rubyTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateDotNetHttpClientCommand(requestSpec HTTPRequestSpec) (string, error) {
	const dotNetTemplate = `
using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

public class Program {
    private static readonly HttpClient client = new HttpClient();

    public static async Task Main(string[] args) {
        var url = "{{ .URL }}{{ if .Request.QueryParams }}?{{ range $i, $p := .Request.QueryParams }}{{ if $i }}&{{ end }}{{ $p.Key }}={{ $p.Value }}{{ end }}{{ end }}";
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

	// Parse and execute the template
	tmpl, err := template.New("dotNetCommand").Funcs(template.FuncMap{
		"titleize": func(s string) string {
			return strings.Title(strings.ToLower(s))
		},
	}).Parse(dotNetTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, requestSpec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}