# Serica Programming Language

**一门全中文的通用编程语言：变量名、关键字、标点均为中文，编写代码不再需要使用英语；设计上编译器以自身语言编写自身，直接产出可执行程序，不经过 C 语言，也不依赖 LLVM。**

Serica is a new programming language designed for systems programming and system-level development.

**话题标签**（20 个，仅英文）：`arm64`、`automatic-reference-counting`、`bare-metal`、`bootstrap`、`chinese`、`chinese-programming`、`cjk`、`compiler`、`concurrency`、`cross-platform`、`general-purpose-language`、`language-design`、`monomorphization`、`native-code`、`non-english`、`programming-language`、`self-hosting`、`systems-programming`、`unicode`、`wasm`
## Features
- A clean, readable syntax with no whitespace-sensitive elements.
 - Designed for systems programming and system level development.
 - Fully typed type system with strong type inference.

## Goals
- Zero-cost abstractions (zero runtime overhead).
 - Fully self-hosting compiler written in Serica.
- Complete cross-platform support including macOS, Linux and Windows.

## Usage
Compile a `.serica` file:
```sh
serica compile main.serica
```

## Development Setup

To build and run Serica, you'll need to:

1. Clone the repository:
```sh
git clone https://github.com/yourusername/Serica.git
cd Serica
```

2. Install dependencies (instructions coming soon).

3. Build the project:
```sh
make build
```

## Development Plan

This is a working implementation of the Serica language, which follows the development plan outlined in `Serica-开发手册.md`:

1. Set up development environment.
2. Implement lexer, parser, and compiler passes.
3. Develop runtime support for all target platforms.
4. Implement standard library functions.

## Contributing

If you'd like to contribute, please read the [contributing guidelines](CONTRIBUTING.md).

## License

Serica is licensed under MIT.