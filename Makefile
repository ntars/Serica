# Makefile for Serica compiler project

CC = gcc
CFLAGS = -Wall -Wextra -std=c99 -g
CXX = g++
CXXFLAGS = -Wall -Wextra -std=c++14
LD = ld

# Source directories and files
SRC_DIR := src
LIB_DIR := lib

# Build directory for intermediate build artifacts
BUILD_DIR := build

# Default target
all: serica

# Build the Serica compiler executable 
serica: build-serica

build-serica:
	@echo "Building Serica compiler..."
	$(CC) $(CFLAGS) -o serica src/main.c

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/* 

.PHONY: all clean build-serica