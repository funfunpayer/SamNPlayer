#pragma once
#include <vector>
#include <string>
#include <cstdint>
#include <optional>

namespace samn {

struct Action {
    int64_t at = 0;
    int pos = 0;
};

struct Script {
    std::vector<Action> actions;
    bool inverted = false;
    int dropped = 0;
    int64_t duration_meta = 0;  // ms, after OFS scaling rules
};

// MaxActionMs: action times above this (48 h) are treated as corrupt.
constexpr int64_t MaxActionMs = 48LL * 3600 * 1000;

// Parse a funscript JSON string. Throws std::runtime_error on hard failures
// (invalid JSON, no actions, all times corrupt). Soft drops corrupt times.
Script Parse(const std::string& json_str);

// Write a minimal funscript JSON (actions + inverted + duration if set).
// Full metadata is not preserved; this is the hot-path subset for M1.
std::string Write(const Script& s);

// Speed-Highlights: intensity = 500 * |Δpos| / |Δt_ms|.
// DefaultMaxIntensity = 400.0 when max_intensity <= 0.
struct SpeedSegment {
    int64_t from_ms = 0;
    int64_t to_ms = 0;
    double intensity = 0.0;
};

std::vector<SpeedSegment> SpeedHighlights(const std::vector<Action>& actions, double max_intensity = 0.0);

}  // namespace samn
