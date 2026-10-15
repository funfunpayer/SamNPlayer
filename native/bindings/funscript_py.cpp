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

    m.def("clean_actions", &samn::CleanActions);
    m.def("write_actions", &samn::WriteActions);
    m.def("max_action_ms", []() { return samn::kMaxActionMs; });
}
