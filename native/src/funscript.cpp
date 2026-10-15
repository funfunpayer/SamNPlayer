#include "funscript.h"
#include <algorithm>
#include <sstream>

namespace samn {

std::vector<Action> CleanActions(std::vector<Action> actions) {
    std::vector<Action> kept;
    kept.reserve(actions.size());
    for (const auto& a : actions) {
        if (a.at >= 0 && a.at <= kMaxActionMs) {
            Action clean = a;
            if (clean.pos < 0) clean.pos = 0;
            else if (clean.pos > 100) clean.pos = 100;
            kept.push_back(clean);
        }
    }
    std::stable_sort(kept.begin(), kept.end(), [](const Action& x, const Action& y) {
        return x.at < y.at;
    });
    return kept;
}

std::string WriteActions(const std::vector<Action>& actions) {
    std::ostringstream os;
    os << "{\n  \"actions\": [";
    for (size_t i = 0; i < actions.size(); ++i) {
        if (i) os << ",";
        os << "\n    {\"at\": " << actions[i].at << ", \"pos\": " << actions[i].pos << "}";
    }
    os << "\n  ]\n}";
    return os.str();
}

}  // namespace samn
