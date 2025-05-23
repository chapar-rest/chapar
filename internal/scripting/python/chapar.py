#!/usr/bin/env python3
import types
import sys
import json
import argparse
import traceback
from flask import Flask, request, jsonify

def create_chapar_module():
    """
    Dynamically create the chapar module with full implementation
    """
    # Create a new module object
    chapar_module = types.ModuleType('chapar')
    chapar_module.__file__ = '<dynamic>'
    chapar_module.__doc__ = """
    chapar module - Interface for interacting with the Chapar application
    """

    # Store environments internally
    environments = {}
    set_environments = {}

    # Environment variable methods
    def get_env(name):
        value = environments.get(name)
        return value

    def set_env(name, value):
        set_environments[name] = value

    def log(message):
        print(f"CHAPAR_LOG: {message}")

    # Assign methods to the module
    chapar_module.get_env = get_env
    chapar_module.set_env = set_env
    chapar_module.log = log
    chapar_module.on_response = None
    chapar_module._environments = environments
    chapar_module._set_environments = set_environments

    # Register the module in sys.modules
    sys.modules['chapar'] = chapar_module
    return chapar_module


# Create the chapar module
chapar = create_chapar_module()
app = Flask(__name__)


@app.route("/health")
def health_check():
    return jsonify({"status": "ok"})


@app.route("/execute-pre-request", methods=["POST"])
def execute_pre_request():
    try:
        data = request.json
        script = data.get("script", "")
        request_data = data.get("requestData", {})
        environments = data.get("environments", {})

        # Update chapar module environments
        chapar._environments.clear()
        chapar._set_environments.clear()
        chapar._environments.update(environments)

        # Prepare execution environment
        globals_dict = {
            "__builtins__": __builtins__,
            "chapar": chapar  # Make chapar available in globals
        }

        locals_dict = {
            "request": request_data,
            "chapar": chapar,  # Also make it available in locals
            "print": print
        }

        # Now execute the actual script
        exec(script, globals_dict, locals_dict)

        # Return the potentially modified data
        return jsonify({
            "requestData": locals_dict["request"],
            "set_environments": chapar._set_environments
        })
    except Exception as e:
        return jsonify({"error": str(e), "traceback": traceback.format_exc()}), 400


@app.route("/execute-post-response", methods=["POST"])
def execute_post_response():
    try:
        data = request.json
        script = data.get("script", "")
        request_data = data.get("requestData", {})
        response_data = data.get("responseData", {})
        environments = data.get("environments", {})

        # Update chapar module environments
        chapar._environments.clear()
        chapar._set_environments.clear()
        chapar._environments.update(environments)

        # Create response object
        response_obj = type("ResponseObject", (), {
            "status_code": response_data.get("statusCode"),
            "headers": response_data.get("headers", {}),
            "text": response_data.get("body", ""),
            "json": lambda self=None: json.loads(response_data.get("body", "{}")),
        })()

        # Prepare execution environment
        globals_dict = {
            "__builtins__": __builtins__,
            "chapar": chapar  # Make chapar available in globals
        }

        locals_dict = {
            "request": request_data,
            "response": response_obj,
            "chapar": chapar,  # Also make it available in locals
            "print": print
        }

        # Reset any callbacks
        chapar.on_response = None

        # Execute the script
        exec(script, globals_dict, locals_dict)

        # If on_response was set, call it
        if chapar.on_response is not None and callable(chapar.on_response):
            chapar.on_response(response_obj)

        # Return the potentially modified data
        return jsonify({
            "environments": chapar._environments,
            "set_environments": chapar._set_environments,
        })
    except Exception as e:
        return jsonify({"error": str(e), "traceback": traceback.format_exc()}), 400


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8090)
    args = parser.parse_args()
    app.run(host="127.0.0.1", port=args.port, debug=False)

