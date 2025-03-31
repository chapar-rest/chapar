<div align="center">
  <img src="./build/appicon.png" alt="Chapar"/>
  <br/>
  <br/>
  <a href="https://img.shields.io/github/v/release/chapar-rest/chapar?include_prereleases" title="Latest Release" rel="nofollow"><img src="https://img.shields.io/github/v/release/chapar-rest/chapar?include_prereleases" alt="Latest Release"></a>
  <a href='https://gophers.slack.com/messages/chapar'><img src='https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=blue' alt='Join us on Slack' /></a>
  <a href='https://www.youtube.com/channel/UCn7EZpdKM8SWE0JcVS3ZXrQ'>
  <img alt="YouTube Channel Subscribers" src="https://img.shields.io/youtube/channel/subscribers/UCn7EZpdKM8SWE0JcVS3ZXrQ">
  </a>
</div>

# Chapar - Native API Testing Tool

Chapar is an upcoming native API testing tool built with GoLang, designed to simplify and expedite the testing process for developers. While still in its early beta phase, Chapar aims to offer a user-friendly experience with support for both HTTP and gRPC protocols with.

## What Chapar means?
Chapar was the institution of the royal mounted couriers in ancient Persia.
The messengers, called Chapar, alternated in stations a day's ride apart along the Royal Road.
The riders were exclusively in the service of the Great King and the network allowed for messages to be transported from Susa to Sardis (2699 km) in nine days; the journey took ninety days on foot.

Herodus described the Chapar as follows:

> There is nothing in the world that travels faster than these Persian couriers. Neither snow, nor rain, nor heat, nor darkness of night prevents these couriers from completing their designated stages with utmost speed.
>
> Herodotus, about 440 BC

## State of the project
Chapar is currently in the early beta phase and under active development, with regular updates and improvements planned to enhance the user experience and functionality.

## Screenshots
<div align="center">
  <img src="screenshots/requests_details.png" alt="Chapar" width="400"/>
  <img src="./screenshots/environments.png" alt="Chapar" width="400"/>
  <img src="./screenshots/post_request.png" alt="Chapar" width="400"/>
  <img src="./screenshots/workspaces.png" alt="Chapar" width="400"/>
  <img src="./screenshots/params.png" alt="Chapar" width="400"/>
  <img src="./screenshots/params.png" alt="Chapar" width="400"/>
  <img src="./screenshots/protofiles.png" alt="Chapar" width="400"/>
  <img src="./screenshots/grpc_request.png" alt="Chapar" width="400"/>
</div>


### Features
* Create and manage workspaces to organize your API endpoints.
* Create and manage environments to store variables and configurations for your API endpoints.
* Create and manage requests to test your API endpoints.
* Send requests with different methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTION,CONNECT).
* Send requests with different content types (JSON, XML, Form, Text, HTML).
* Send requests with different authentication methods (Basic, Bearer, API Key, No Auth).
* Send requests with different body types (Form, Raw, Binary).
* Set environment variables from the response of the request using JSONPath.
* Dark mode support.
* Data is stored locally on your machine. and no data is sent to any server.
* Import collections and requests from Postman.
* Support GRPC protocol.
* Support for grpc reflection and proto files.
* Load sample request structure of given grpc method.
* Chaining requests with Pre/Post request option.

### Roadmap
* Support WebSocket, GraphQL protocol.
* Python as a scripting language for pre-request and post-request scripts.
* Support for tunneling to servers and kube clusters as pre request actions.

### Getting Started
To Get started with Chapar, you can download the latest release from the [releases page](https://github.com/chapar-rest/chapar/releases).
Their you can find the latest release for your operating system.

#### Install on macOS
On macOS, you can either download the latest release or install Chapar from Apple's App Store.

<a href="https://apps.apple.com/us/app/chapar-rest/id6673918597?mt=12&itscg=30200&itsct=apps_box_badge&mttnsubad=6673918597" style="display: inline-block;">
<img src="https://toolbox.marketingtools.apple.com/api/v2/badges/download-on-the-app-store/black/en-us?releaseDate=1743379200" alt="Download on the App Store" style="width: 150px; height: 50px; vertical-align: middle; object-fit: contain;" />
</a>
<br/><br/>
Note that the App Store version is running in a sandbox environment and if you are already using the downloaded
or custom build version, you need to copy your data to the sandbox environment. you can do it by running the following command:

```bash
cp -r $HOME/.config/chapar $HOME/Library/Containers/rest.chapar.app/Data/.config
```
or make a symlink to the sandbox environment:
```bash
ln -s $HOME/.config/chapar $HOME/Library/Containers/rest.chapar.app/Data/.config
```

#### Install From AUR
On Arch-based distros, you can install Chapar from the AUR using your favorite AUR helper:
```bash
yay -S chapar-bin
```
Please note that AUR package is maintained by a community contributor. (@Monirzadeh ) may not be up to date with the latest release.

#### Install From Source
To install Chapar from source, clone the repository install the dependencies, and run the application using the following commands:
```bash
git clone https://github.com/chapar-rest/chapar.git
cd chapar
go build -o chapar .
```

## Dependencies
If you want to build the project from source, you need to install the following dependencies:
Chapar is built using [Gio](https://gioui.org) library so you need to install the following dependencies to build the project:

for linux follow instructions in [gio linux](https://gioui.org/doc/install/linux)
for macOS follow instructions in [gio macos](https://gioui.org/doc/install/macos)


### Contributing
We welcome contributions from the community once the early beta is released! If you have ideas, feedback, or wish to contribute, please open an issue or submit a pull request.

### Support
You can support the development of Chapar by starring the repository, sharing it with your friends, and contributing to the project.
Also you can support the project by donating to the project's wallet.

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/mohsen.mirzakhani)
