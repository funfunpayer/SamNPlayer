#include <pybind11/pybind11.h>
#include <pybind11/stl.h>
#include "funscript.h"

namespace py = pybind11;

PYBIND11_MODULE(funscript_native, m) {
    py::class_<samn::Action>(m, "Action")
        .def(py::init<>())
        .def(py::init([](int64_t at, int pos) { return samn::Action{at, pos}; }))
        .def_readwrite("at", &samn::Action::at)
        .def_readwrite("pos", &samn::Action::pos);

    py::class_<samn::Script>(m, "Script")
        .def(py::init<>())
        .def_readwrite("actions", &samn::Script::actions)
        .def_readwrite("inverted", &samn::Script::inverted)
        .def_readwrite("dropped", &samn::Script::dropped)
        .def_readwrite("duration_meta", &samn::Script::duration_meta)
        .def_readwrite("metadata_json", &samn::Script::metadata_json);

    m.def("parse", &samn::Parse);
    m.def("write", &samn::Write);
}
