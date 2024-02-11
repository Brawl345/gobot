# Simple Google Cloud Function that acts as a proxy for requests to the Gemini API.
# Make sure you deploy the Cloud Function in a US datacenter

import functions_framework

@functions_framework.http
def make_request(request):
    if request.method != 'POST':
        return 'Invalid request method', 405

    import requests

    base_url = "https://generativelanguage.googleapis.com/v1beta/models"
    get_params = dict(request.args)
    model_name = get_params.pop('model', 'gemini-pro')
    function_name = get_params.pop('function', 'generateContent')
    url_with_params = f"{base_url}/{model_name}:{function_name}"

    if get_params:
        url_with_params += '?' + '&'.join([f"{key}={value}" for key, value in get_params.items()])

    post_data = request.get_json(silent=True)

    response = requests.post(url_with_params, json=post_data)
    print(response.text)
    return response.text, response.status_code