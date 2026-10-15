#pragma once
#include <vector>
#include <cstdint>
#include <string>

namespace samn {

struct Action {
    int64_t at;
    int pos;
};

struct StrokeFilter {
    std::string smooth;     // "", "pchip", "makima"
    int64_t step_ms = 100;
    double max_speed = 0.0; // units per second, 0 = no limit
    int64_t idle_fill_ms = 0;
};

bool filter_active(const StrokeFilter& f);

std::string normalize_smooth(const std::string& s);

// Port of funscript.ApplyStrokeFilter
std::vector<Action> apply_stroke_filter(const std::vector<Action>& actions, const StrokeFilter& f);

}  // namespace samn
