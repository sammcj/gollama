# Gollama

![](gollama-logo.png)

Gollama is a macOS / Linux tool for managing Ollama models.

It provides a TUI (Text User Interface) for listing, inspecting, deleting, copying, and pushing Ollama models as well as optionally linking them to LM Studio*.

The application allows users to interactively select models, sort, filter, edit, run, unload and perform actions on them using hotkeys.

![](screenshots/gollama-v1.0.0.jpg)

## Table of Contents

- [Gollama](#gollama)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Installation](#installation)
  - [Usage](#usage)
    - [Key Bindings](#key-bindings)
      - [Top](#top)
      - [Inspect](#inspect)
      - [Command-line Options](#command-line-options)
  - [Configuration](#configuration)
  - [Installation and build from source](#installation-and-build-from-source)
  - [Logging](#logging)
  - [Contributing](#contributing)
  - [Acknowledgements](#acknowledgements)
  - [License](#license)

## Features

The project started off as a rewrite of my [llamalink](https://smcleod.net/2024/03/llamalink-ollama-to-lm-studio-llm-model-linker/) project, but I decided to expand it to include more features and make it more user-friendly.

It's in active development, so there are some bugs and missing features, however I'm finding it useful for managing my models every day, especially for cleaning up old models.

- List available models
- Display metadata such as size, quantisation level, model family, and modified date
- Edit / update a model's Modelfile
- Sort models by name, size, modification date, quantisation level, family etc
- Select and delete models
- Run and unload models
- Inspect model for additional details
- Calculate approximate vRAM usage for a model
- Link models to LM Studio **Note: This is currently broken on the latest LM-Studio versions, see [#82](https://github.com/sammcj/gollama/issues/82)**
- Copy / rename models
- Push models to a registry
- Show running models
- Has some cool bugs

## Installation

From go:

```shell
go install github.com/sammcj/gollama@HEAD
```

From Github:

Download the most recent release from the [releases page](https://github.com/sammcj/gollama/releases) and extract the binary to a directory in your PATH.

e.g. `zip -d gollama*.zip -d gollama && mv gollama /usr/local/bin`

## Usage

To run the `gollama` application, use the following command:

```sh
gollama
```

_Tip_: I like to alias gollama to `g` for quick access:

```shell
echo "alias g=gollama" >> ~/.zshrc
```

### Key Bindings

- `Space`: Select
- `Enter`: Run model (Ollama run)
- `i`: Inspect model
- `t`: Top (show running models)
- `D`: Delete model
- `e`: Edit model **new**
- `c`: Copy model
- `U`: Unload all models
- `p`: Pull an existing model **new**
- `g`: Pull (get) new model **new**
- `P`: Push model
- `n`: Sort by name
- `s`: Sort by size
- `m`: Sort by modified
- `k`: Sort by quantisation
- `f`: Sort by family
- `l`: Link model to LM Studio
- `L`: Link all models to LM Studio
- `r`: Rename model _**(Work in progress)**_
- `q`: Quit

#### Top

Top (`t`)

![](screenshots/gollama-top.jpg)

#### Inspect

Inspect (`i`)

![](screenshots/gollama-inspect.png)

#### Command-line Options

- `-l`: List all available Ollama models and exit
- `-L`: Link all available Ollama models to LM Studio and exit **new**
- `-s <search term>`: Search for models by name **new**
  - OR operator (`'term1|term2'`) returns models that match either term
  - AND operator (`'term1&term2'`) returns models that match both terms
- `-e <model>`: Edit the Modelfile for a model **new**
- `-ollama-dir`: Custom Ollama models directory
- `-lm-dir`: Custom LM Studio models directory
- `-cleanup`: Remove all symlinked models and empty directories and exit
- `-no-cleanup`: Don't cleanup broken symlinks
- `-u`: Unload all running models
- `-v`: Print the version and exit
- `--vram`: Estimate vRAM usage for a model
  - `--model`: Model ID for vRAM estimation
  - `--quant`: Quantisation type (e.g., q4_k_m) or bits per weight (e.g., 5.0)
  - `--context`: Context length for vRAM estimation
  - `--kvcache`: KV cache quantisation: fp16, q8_0, or q4_0
  - `--memory`: Available memory in GB for context calculation
  - `--quanttype`: Quantisation type: gguf or exl2

##### Simple model listing

Gollama can also be called with `-l` to list models without the TUI.

```shell
gollama -l
```

List (`gollama -l`):

![](screenshots/cli-list.jpg)

##### Edit

Gollama can be called with `-e` to edit the Modelfile for a model.

```shell
gollama -e my-model
```

##### Search

Gollama can be called with `-s` to search for models by name.

```shell
gollama -s my-model # returns models that contain 'my-model'

gollama -s 'my-model|my-other-model' # returns models that contain either 'my-model' or 'my-other-model'

gollama -s 'my-model&instruct' # returns models that contain both 'my-model' and 'instruct'
```

##### vRAM Estimation

- Calculate vRAM usage for a given model configuration
- Determine maximum context length for a given vRAM constraint
- Find the best quantisation setting for a given vRAM and context constraint
- Support for different k/v cache quantisation options (fp16, q8_0, q4_0)

To estimate VRAM usage:

```shell
gollama --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant q4_k_m --context 2048 --kvcache q4_0 # For GGUF models
gollama --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant 5.0 --context 2048 --kvcache q4_0 # For exl2 models
# Estimated VRAM usage: 5.35 GB
```

To calculate maximum context for a given memory constraint:

```shell
gollama --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant q4_k_m --memory 6 --kvcache q8_0 # For GGUF models
gollama --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --bpw 5.0 --memory 6 --kvcache q8_0 # For exl2 models
# Maximum context for 6.00 GB of memory: 5069
```

To find the best BPW:

```shell
gollama --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --memory 6 --quanttype gguf
# Best BPW for 6.00 GB of memory: IQ3_S
```

The vRAM estimator works by:

1. Fetching the model configuration from Hugging Face (if not cached locally)
2. Calculating the memory requirements for model parameters, activations, and KV cache
3. Adjusting calculations based on the specified quantisation settings
4. Performing binary and linear searches to optimize for context length or quantisation settings

## Configuration

Gollama uses a JSON configuration file located at `~/.config/gollama/config.json`. The configuration file includes options for sorting, columns, API keys, log levels etc...

Example configuration:

```json
{
  "default_sort": "modified",
  "columns": [
    "Name",
    "Size",
    "Quant",
    "Family",
    "Modified",
    "ID"
  ],
  "ollama_api_key": "",
  "ollama_api_url": "http://localhost:11434",
  "lm_studio_file_paths": "",
  "log_level": "info",
  "log_file_path": "/Users/username/.config/gollama/gollama.log",
  "sort_order": "Size",
  "strip_string": "my-private-registry.internal/",
  "editor": "",
  "docker_container": ""
}
```

- `strip_string` can be used to remove a prefix from model names as they are displayed in the TUI. This can be useful if you have a common prefix such as a private registry that you want to remove for display purposes.
- `docker_container` - **experimental** - if set, gollama will attempt to perform any run operations inside the specified container.
- `editor` - **experimental** - if set, gollama will use this editor to open the Modelfile for editing.

## Installation and build from source

1. Clone the repository:

    ```shell
    git clone https://github.com/sammcj/gollama.git
    cd gollama
    ```

2. Build:

    ```shell
    go get
    make build
    ```

3. Run:

    ```shell
    ./gollama
    ```

## Logging

Logs can be found in the `gollama.log` which is stored in `$HOME/.config/gollama/gollama.log` by default.
The log level can be set in the configuration file.

## Contributing

Contributions are welcome!
Please fork the repository and create a pull request with your changes.

<!-- readme: contributors -start -->
<table>
	<tbody>
		<tr>
            <td align="center">
                <a href="https://github.com/sammcj">
                    <img src="https://avatars.githubusercontent.com/u/862951?v=4" width="50;" alt="sammcj"/>
                    <br />
                    <sub><b>Sam</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/josekasna">
                    <img src="https://avatars.githubusercontent.com/u/138180151?v=4" width="50;" alt="josekasna"/>
                    <br />
                    <sub><b>Jose Almaraz</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/jralmaraz">
                    <img src="https://avatars.githubusercontent.com/u/13877691?v=4" width="50;" alt="jralmaraz"/>
                    <br />
                    <sub><b>Jose Roberto Almaraz</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/anrgct">
                    <img src="https://avatars.githubusercontent.com/u/16172523?v=4" width="50;" alt="anrgct"/>
                    <br />
                    <sub><b>anrgct</b></sub>
                </a>
            </td>
		</tr>
	<tbody>
</table>
<!-- readme: contributors -end -->

## Acknowledgements

- [Ollama](https://ollama.com/)
- [Llama.cpp](https://github.com/ggerganov/llama.cpp)
- [Charmbracelet](https://charm.sh/)

Thank you to folks such as Matt Williams, Fahd Mirza and AI Code King for giving this a shot and providing feedback.

[![AI Code King - Easiest & Interactive way to Manage & Run Ollama Models Locally](https://img.youtube.com/vi/T4uiTnacyhI/0.jpg)](https://www.youtube.com/watch?v=T4uiTnacyhI)
[![Matt Williams - My favourite way to run Ollama: Gollama](https://img.youtube.com/vi/OCXuYm6LKgE/0.jpg)](https://www.youtube.com/watch?v=OCXuYm6LKgE)
[![Fahd Mirza - Gollama - Manage Ollama Models Locally](https://img.youtube.com/vi/24yqFrQV-4Q/0.jpg)](https://www.youtube.com/watch?v=24yqFrQV-4Q)

## License

Copyright © 2024 Sam McLeod

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
