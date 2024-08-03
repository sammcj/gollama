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
- `l`: Link model to LM Studio **Note: This is currently broken on the latest LM-Studio versions, see [#82](https://github.com/sammcj/gollama/issues/82)**
- `L`: Link all models to LM Studio *^
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
- `-L`: Link all available Ollama models to LM Studio and exit **Note: This is currently broken on the latest LM-Studio versions, see [#82](https://github.com/sammcj/gollama/issues/82)**
- `-s <search term>`: Search for models by name
  - OR operator (`'term1|term2'`) returns models that match either term
  - AND operator (`'term1&term2'`) returns models that match both terms
- `-e <model>`: Edit the Modelfile for a model
- `-ollama-dir`: Custom Ollama models directory
- `-lm-dir`: Custom LM Studio models directory
- `-cleanup`: Remove all symlinked models and empty directories and exit
- `-no-cleanup`: Don't cleanup broken symlinks
- `-u`: Unload all running models
- `-v`: Print the version and exit
- `--vram`: Estimate vRAM usage for a huggingface model ID (e.g. `NousResearch/Hermes-2-Theta-Llama-3-8B`) **new**
  - `--fits`: Available memory in GB for context calculation (e.g. `6` for 6GB)

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

Gollama includes a comprehensive vRAM estimation feature:

- Calculate vRAM usage for a given huggingface model ID (working on adding Ollama models)
- Determine maximum context length for a given vRAM constraint
- Find the best quantisation setting for a given vRAM and context constraint
- Shows estimates for different k/v cache quantisation options (fp16, q8_0, q4_0)
- Automatic detection of available CUDA vRAM (**coming soon!**) or system RAM

![](screenshots/vram.png)

To estimate (v)RAM usage:

```shell
gollama --vram NousResearch/Hermes-2-Theta-Llama-3-8B
```

To find the best quantisation type for a given memory constraint (e.g. 6GB):

```shell
gollama --vram NousResearch/Hermes-2-Theta-Llama-3-8B --fits 6

gollama --vram NousResearch/Hermes-2-Theta-Llama-3-8B --fits 6                                         (tableU)
📊 VRAM Estimation for Model: NousResearch/Hermes-2-Theta-Llama-3-8B

| QUANT/CTX | BPW  | 2K  | 8K   | 16K             | 32K             | 49K             | 64K             |
| --------- | ---- | --- | ---- | --------------- | --------------- | --------------- | --------------- |
| IQ1_S     | 1.56 | 2.4 | 3.8  | 5.7(4.7,4.2)    | 9.5(7.5,6.5)    | 13.3(10.3,8.8)  | 17.1(13.1,11.1) |
| IQ2_XXS   | 2.06 | 2.9 | 4.3  | 6.3(5.3,4.8)    | 10.1(8.1,7.1)   | 13.9(10.9,9.4)  | 17.8(13.8,11.8) |
| IQ2_XS    | 2.31 | 3.1 | 4.6  | 6.5(5.5,5.0)    | 10.4(8.4,7.4)   | 14.2(11.2,9.8)  | 18.1(14.1,12.1) |
| IQ2_S     | 2.50 | 3.3 | 4.8  | 6.7(5.7,5.2)    | 10.6(8.6,7.6)   | 14.5(11.5,10.0) | 18.4(14.4,12.4) |
| IQ2_M     | 2.70 | 3.5 | 5.0  | 6.9(5.9,5.4)    | 10.8(8.8,7.8)   | 14.7(11.7,10.2) | 18.6(14.6,12.6) |
| IQ3_XXS   | 3.06 | 3.8 | 5.3  | 7.3(6.3,5.8)    | 11.2(9.2,8.2)   | 15.2(12.2,10.7) | 19.1(15.1,13.1) |
| IQ3_XS    | 3.30 | 4.1 | 5.5  | 7.5(6.5,6.0)    | 11.5(9.5,8.5)   | 15.5(12.5,11.0) | 19.4(15.4,13.4) |
| Q2_K      | 3.35 | 4.1 | 5.6  | 7.6(6.6,6.1)    | 11.6(9.6,8.6)   | 15.5(12.5,11.0) | 19.5(15.5,13.5) |
| Q3_K_S    | 3.50 | 4.3 | 5.8  | 7.7(6.7,6.2)    | 11.7(9.7,8.7)   | 15.7(12.7,11.2) | 19.7(15.7,13.7) |
| IQ3_S     | 3.50 | 4.3 | 5.8  | 7.7(6.7,6.2)    | 11.7(9.7,8.7)   | 15.7(12.7,11.2) | 19.7(15.7,13.7) |
| IQ3_M     | 3.70 | 4.5 | 6.0  | 8.0(7.0,6.5)    | 11.9(9.9,8.9)   | 15.9(12.9,11.4) | 20.0(16.0,14.0) |
| Q3_K_M    | 3.91 | 4.7 | 6.2  | 8.2(7.2,6.7)    | 12.2(10.2,9.2)  | 16.2(13.2,11.7) | 20.2(16.2,14.2) |
| IQ4_XS    | 4.25 | 5.0 | 6.5  | 8.5(7.5,7.0)    | 12.6(10.6,9.6)  | 16.6(13.6,12.1) | 20.7(16.7,14.7) |
| Q3_K_L    | 4.27 | 5.0 | 6.5  | 8.5(7.5,7.0)    | 12.6(10.6,9.6)  | 16.6(13.7,12.2) | 20.7(16.7,14.7) |
| IQ4_NL    | 4.50 | 5.2 | 6.7  | 8.8(7.8,7.3)    | 12.9(10.9,9.9)  | 16.9(13.9,12.4) | 21.0(17.0,15.0) |
| Q4_0      | 4.55 | 5.2 | 6.8  | 8.8(7.8,7.3)    | 12.9(10.9,9.9)  | 17.0(14.0,12.5) | 21.1(17.1,15.1) |
| Q4_K_S    | 4.58 | 5.3 | 6.8  | 8.9(7.9,7.4)    | 12.9(10.9,9.9)  | 17.0(14.0,12.5) | 21.1(17.1,15.1) |
| Q4_K_M    | 4.85 | 5.5 | 7.1  | 9.1(8.1,7.6)    | 13.2(11.2,10.2) | 17.4(14.4,12.9) | 21.5(17.5,15.5) |
| Q4_K_L    | 4.90 | 5.6 | 7.1  | 9.2(8.2,7.7)    | 13.3(11.3,10.3) | 17.4(14.4,12.9) | 21.6(17.6,15.6) |
| Q5_K_S    | 5.54 | 6.2 | 7.8  | 9.8(8.8,8.3)    | 14.0(12.0,11.0) | 18.2(15.2,13.7) | 22.4(18.4,16.4) |
| Q5_0      | 5.54 | 6.2 | 7.8  | 9.8(8.8,8.3)    | 14.0(12.0,11.0) | 18.2(15.2,13.7) | 22.4(18.4,16.4) |
| Q5_K_M    | 5.69 | 6.3 | 7.9  | 10.0(9.0,8.5)   | 14.2(12.2,11.2) | 18.4(15.4,13.9) | 22.6(18.6,16.6) |
| Q5_K_L    | 5.75 | 6.4 | 8.0  | 10.1(9.1,8.6)   | 14.3(12.3,11.3) | 18.5(15.5,14.0) | 22.7(18.7,16.7) |
| Q6_K      | 6.59 | 7.2 | 9.0  | 11.4(10.4,9.9)  | 16.2(14.2,13.2) | 21.0(18.0,16.5) | 25.8(21.8,19.8) |
| Q8_0      | 8.50 | 9.1 | 10.9 | 13.4(12.4,11.9) | 18.4(16.4,15.4) | 23.4(20.4,18.9) | 28.3(24.3,22.3) |
```

This will display a table showing vRAM usage for various quantisation types and context sizes.

The vRAM estimator works by:

1. Fetching the model configuration from Hugging Face (if not cached locally)
2. Calculating the memory requirements for model parameters, activations, and KV cache
3. Adjusting calculations based on the specified quantisation settings
4. Performing binary and linear searches to optimize for context length or quantisation settings

Note: The estimator will attempt to use CUDA vRAM if available, otherwise it will fall back to system RAM for calculations.

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
