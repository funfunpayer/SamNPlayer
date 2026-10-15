cmake_minimum_required(VERSION 3.16)
project(samn_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# pybind11
find_package(pybind11 REQUIRED)

# Beat bar module
pybind11_add_module(beatbar_native src/beatbar.cpp bindings/beatbar_py.cpp)

target_include_directories(beatbar_native PRIVATE include)

# Install
install(TARGETS beatbar_native DESTINATION .)
