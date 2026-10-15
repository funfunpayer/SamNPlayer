#include <pybind11/pybind11.h>
#include <pybind11/stl.h>
#include "stroke_filter.h"

namespace py = pybind11;

PYBIND11_MODULE(stroke_filter_native, m) {
    m.doc() = "SamNPlayer stroke filter (parity with funscript.ApplyStrokeFilter)";

    py::class_<samn::Action>(m, "Action")
        .def(py::init<>())
        .def(py::init<int64_t, int>())
        .def_readwrite("at", &samn::Action::at)
        .def_readwrite("pos", &samn::Action::pos)
        .def("__repr__", [](const samn::Action& a) {
            return "<Action at=" + std::to_string(a.at) + " pos=" + std::to_string(a.pos) + ">";
        });

    py::class_<samn::StrokeFilter>(m, "StrokeFilter")
        .def(py::init<>())
        .def_readwrite("smooth", &samn::StrokeFilter::smooth)
        .def_readwrite("step_ms", &samn::StrokeFilter::step_ms)
        .def_readwrite("max_speed", &samn::StrokeFilter::max_speed)
        .def_readwrite("idle_fill_ms", &samn::StrokeFilter::idle_fill_ms);

    m.def("apply_stroke_filter", &samn::apply_stroke_filter,
          "Apply stroke filter (parity with Go funscript.ApplyStrokeFilter)",
          py::arg("actions"), py::arg("filter"));
}
