cmake_minimum_required(VERSION 3.16)
project(samn_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# pybind11
find_package(pybind11 REQUIRED)

# Beat bar module
pybind11_add_module(beatbar_native src/beatbar.cpp bindings/beatbar_py.cpp)
target_include_directories(beatbar_native PRIVATE include)

# Stroke filter module
pybind11_add_module(stroke_filter_native src/stroke_filter.cpp bindings/stroke_filter_py.cpp)
target_include_directories(stroke_filter_native PRIVATE include)

# Beat strokes module
pybind11_add_module(beat_strokes_native src/beat_strokes.cpp bindings/beat_strokes_py.cpp)
target_include_directories(beat_strokes_native PRIVATE include)

# Install
install(TARGETS beatbar_native stroke_filter_native beat_strokes_native DESTINATION .)
