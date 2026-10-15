cmake_minimum_required(VERSION 3.16)
project(samn_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(FetchContent)
FetchContent_Declare(
  nlohmann_json
  URL https://github.com/nlohmann/json/releases/download/v3.11.3/json.tar.xz
  URL_HASH SHA256=d6c65aca6b1ed68e7a182f4757257b107ae403032760ed6ef121c9d55e81757d
)
FetchContent_MakeAvailable(nlohmann_json)

find_package(pybind11 REQUIRED)

# Beat bar module
pybind11_add_module(beatbar_native src/beatbar.cpp bindings/beatbar_py.cpp)
target_include_directories(beatbar_native PRIVATE include)

# Stroke filter module
pybind11_add_module(stroke_filter_native src/stroke_filter.cpp bindings/stroke_filter_py.cpp)
target_include_directories(stroke_filter_native PRIVATE include)

# Funscript parse/write (core)
pybind11_add_module(funscript_native src/funscript.cpp bindings/funscript_py.cpp)
target_include_directories(funscript_native PRIVATE include)
target_link_libraries(funscript_native PRIVATE nlohmann_json::nlohmann_json)

# Install
install(TARGETS beatbar_native stroke_filter_native funscript_native DESTINATION .)
