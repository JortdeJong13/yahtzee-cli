# Opponent investigation and difficulty design

Implemented for v0.4.1 after investigation on 2 October 2026 against `a5b7c37`. The initial investigation used temporary Go test overlays; the decision checks and calibration harness now live in the repository. The earlier measurements below record how the profiles were selected.

The reported behavior is reproducible. Normal can knowingly discard one of three matching fives because its difficulty is partly implemented by choosing worse moves at random. A deterministic opponent with less foresight can produce a similar average score without that mechanism.

In v0.4.0, Normal searched all remaining rolls and used the same future-score estimate as Expert. It then had a 25% chance to choose another action within four estimated points of the best action. That was a chance to enter the alternative-selection branch; it was not the probability of any particular bad move. This layer has been removed.

A real `Game.PlayOpponent` call with game seed 29, only Fives open, upper subtotal zero, and the Yahtzee box scored zero produced this trace:

```text
First roll:     [5 5 3 2 5]
Chosen locks:   [true true false false false]
```

Normal kept the first two fives and rerolled the third. Keeping all three would produce an expected final Fives score of 18.06; keeping two and then playing optimally would produce 14.58. The loss of 3.47 points fits inside the four-point allowance. The evaluator itself ranks keeping all three first.

The lock mapping passed all 4,368 keep-to-die mappings across the 252 unordered five-dice outcomes. Reroll outcome weights sum to `6^n` for every reroll count from zero through five. This reproduced decision is caused by action selection, rather than an indexing error.

Easy in v0.4.0 had a separate scoring problem. With upper subtotal 48, Fives and Chance open, and a final roll of `[5 5 5 6 6]`, it chose Chance for 27. Fives earns 15 plus the upper bonus of 35 immediately, for 50. All profiles now count that earned bonus through the same scoring transition used by the game.

Expert searches the current turn exactly under its evaluation function, but its estimate of subsequent turns is approximate. The upper and lower sections are solved separately. The lower table is identical for a Yahtzee box containing zero or 50, because the generator omits future 100-point Yahtzee bonuses. The live evaluator does count a bonus Yahtzee when scoring the current dice. The distinction matters when describing Expert and when later improving its future estimates.

Not every split of matching dice is wrong. For the same three-fives roll with only straights open, the deterministic evaluator keeps `[5 2 3]`. Tests must include the scorecard and remaining rolls; they should not impose a general rule to keep every matching die.

Keep one evaluator and the existing scoring rules. Every level chooses the highest-value legal action under its own evaluation. Remove the deliberate mistake selection and its separate random source. The dice provide variation between games.

Use two fixed controls: remaining-reroll search depth and how strongly the evaluator values subsequent scorecard opportunities.

| Level | Maximum rerolls considered | Future weight |
| --- | ---: | ---: |
| Easy | 1 | 0 |
| Normal | 2, or fewer if fewer remain | 0.4 |
| Expert | 2, or fewer if fewer remain | 1 |

The future weight is a degree of future awareness, not a literal count of turns searched. A literal turn-depth implementation would need a different continuation calculation. That extra machinery is unnecessary for the first change.

Compute a category's value as:

```text
earnedNow = category points + newly earned upper bonus + current extra Yahtzee bonus
future(card) = existing continuation estimate(card) - already earned upper bonus(card)
value = earnedNow + futureWeight * (future(nextCard) - future(currentCard))
```

Removing the already earned upper bonus from the future estimate prevents double counting. A weight of one reproduces Expert's current valuation; zero still counts all immediate points correctly. Lower weights soften the estimated cost of consuming useful scorecard slots. For example, a shallow player may spend Chance early because it values today's score more than preserving that slot for later.

These profiles deliberately keep difficulty out of rule enforcement. Joker placement, bonus eligibility, filled categories, and earned points must be identical at every level. Use the same scoring transition when constructing the hypothetical next card and when actually scoring.

For mathematically tied actions, prefer scoring over a no-op reroll, then reroll fewer dice, then use a fixed ordering. Compare held face counts rather than die positions. A tiny numerical tolerance can resolve floating-point ties; do not reintroduce a multi-point allowance for inferior actions.

Do not add configuration, strategy interfaces, dependencies, or a full-game solver. Update the README and AGENTS.md behavior descriptions when the implementation changes.

Fresh baseline runs used 1,000 complete solo opponent games per policy, with actual `Game.Roll` and `Game.Score` transitions. Human play was excluded so it could not consume dice randomness. The dice generator was reset per turn, giving all policies the same first roll on each corresponding turn. Later rolls can differ because policies consume randomness differently. Prototypes retained the existing tie ordering; the proposed tie preference remains to be tested.

| Current or diagnostic policy | Mean | Approximate 95% margin | Upper bonus rate |
| --- | ---: | ---: | ---: |
| Current Easy | 212.904 | ±2.587 | 2.1% |
| Current Normal | 233.030 | ±3.543 | 54.3% |
| Current Expert | 250.494 | ±3.784 | 63.3% |
| Full-turn immediate scoring, uncorrected | 215.516 | ±2.916 | 2.4% |
| One-reroll search with full continuation estimate, deterministic | 244.378 | ±3.528 | 60.2% |

Increasing reroll depth alone barely improves the immediate-score policy. Adding the full future estimate raises its strength substantially. There are only two meaningful reroll depths after the first roll, so depth alone gives coarse steps and does not naturally land Normal near 235.

A 500-game calibration pilot used corrected immediate bonuses and deterministic full-turn search:

| Future weight | Mean | Approximate 95% margin |
| ---: | ---: | ---: |
| 0 | 215.542 | ±4.056 |
| 0.25 | 225.810 | ±4.414 |
| 0.5 | 239.426 | ±4.588 |
| 0.75 | 248.422 | ±5.307 |
| 1 | 251.322 | ±5.354 |

Weight 0.4 was selected as a candidate between the 0.25 and 0.5 results. A separate 1,000-game validation set, unused in that pilot, produced:

| Proposed level | Mean | Approximate 95% margin | Median | 10th–90th percentile | Upper bonus rate |
| --- | ---: | ---: | ---: | --- | ---: |
| Easy | 214.057 | ±2.740 | 207 | 172–261 | 1.8% |
| Normal | 234.737 | ±3.302 | 222 | 186–280 | 24.9% |
| Expert | 249.053 | ±3.607 | 244 | 190–308 | 64.2% |

The candidate Normal reaches roughly 235 without forced mistakes. Expert's policy is unchanged; its different sample mean reflects different dice seeds. These are estimates, rather than guaranteed scores or proof of optimal play. The proposed profiles passed the triple-fives and immediate-upper-bonus cases. That does not establish that every decision will appear sensible.

For reproduction, baseline game IDs were 1–1,000, pilot IDs 1–500, and validation IDs 10,001–11,000. Each turn used dice seed `gameID*100 + turn`, with turns numbered zero through twelve. Current policies initialized their decision RNG through `NewWithSeedAndDifficulty(gameID, difficulty)`. Means include all upper and Yahtzee bonuses from the final scorecard. Margins are `1.96 * sampleSD / sqrt(n)`. Runs used Go 1.27.1 on macOS arm64.

The harness uses ordinary Go tests. Fast decision and reference checks run with `go test ./...`. Long simulations require the `botcheck` build tag:

```sh
go test -tags botcheck ./internal/game -run TestOpponentCalibration -count=1 -v
```

The default is 2,000 games per profile, starting at seed 10,001. To explore weights on a small calibration set:

```sh
go test -tags botcheck ./internal/game -run TestOpponentCalibration -count=1 -v -opponent-games=500 -opponent-seed=1 -opponent-sweep
```

Choose different seeds for validation. The harness uses fixed dice by turn, roll, and physical index; each profile receives the same dice scenarios even when it holds different dice. It invokes `Game.PlayOpponent` for the released profiles, and actual roll and score transitions for experimental weights. It checks completed cards, target bands, and paired score differences. The ordered-dice reference solver validates the current turn under the same approximate future model; it does not establish full-game optimality.

1. **Decision examples.** Store the full scorecard, dice, remaining rolls, and acceptable action or tied actions. Start with the reproduced three-fives case, the immediate upper bonus, Joker placement after Yahtzee zero and 50, preserving a made scoring combination where appropriate, and counterexamples where breaking a group is correct. For every failure, print the chosen action, ranked alternatives, and values. Add newly reported odd choices as replayable examples.
2. **Independent correctness checks.** Retain exhaustive keep mapping and outcome-weight checks. Check permutation invariance of held face counts, deterministic choices, and legal scoring. Compare one-category endgames with hand-derived expectations, such as the Fives calculation above. Add a small independent exhaustive solver for selected one- and two-category endgames, including upper bonus and Joker states, rather than treating Expert's heuristic as an oracle.
3. **Calibration simulations.** Run whole games through actual game transitions. Measure mean, uncertainty, median, score percentiles, upper bonus rate, Yahtzee rate, zeroed categories, and thinking time. Exclude UI pauses and player statistics. Record seeds and profiles alongside each result.

For the permanent simulation harness, generate dice by `(game seed, turn, roll, die index)`, including values for held dice that will be ignored. That keeps later dice comparable even when policies reroll different counts. Keep this test-only; the opponent must not receive future dice or seed information. Use paired per-game score differences when comparing policies on the same scenarios.

Suggested workflow:

- Agree on approximate score bands, such as Easy 205–220 and Normal 230–240, while keeping Expert at its strongest supported policy. Earlier rough targets were around 210, 235, and 254; the current fresh Expert samples are around 249–250.
- Require the decision and rule checks to pass before considering average score.
- Sweep a small grid of depths and future weights on a fixed calibration set. Do not assume increasing depth or weight always improves measured score.
- Validate selected profiles on separate seeds. Use several thousand games for final tuning, increasing the sample only when uncertainty affects a decision. A 1,000-game sample here has a mean uncertainty of roughly three to four points.
- Check strength ordering, score bands, and decision examples together. Reject a profile that hits the mean by introducing implausible choices. Avoid exact-mean assertions in routine tests.
- Keep long simulations behind an explicit test name or build tag. Finish an implementation with a real terminal game at each level, plus the required formatting, vet, and build checks.

After this harness exists, audit future Yahtzee bonuses and the upper-bonus approximation separately, then recalibrate if those estimates change. Keep a stronger future model available to Normal through its weight as well; difficulty should not depend on inconsistent rules.

For the initial investigation, mapping checks, probability weights, replayed decisions, and temporary prototype checks passed. `go vet ./...` and `go build ./...` passed. That design stage did not include an interactive terminal game; implementation validation includes the actual executable at every level. The existing table generator remains behind its separate `tablegen` build tag.

Release validation for v0.4.1 used 2,000 new games per level, with game seeds 20,001–22,000 and fixed dice for every physical index. The released profiles, including their tie preferences, produced:

| Level | Mean | Approximate 95% margin | Median | 10th–90th percentile | Upper bonus rate | Yahtzee rate |
| --- | ---: | ---: | ---: | --- | ---: | ---: |
| Easy | 214.814 | ±1.948 | 207 | 173–262 | 3.4% | 27.6% |
| Normal | 235.499 | ±2.328 | 224 | 186–283 | 27.2% | 28.9% |
| Expert | 249.550 | ±2.644 | 243 | 190–313 | 64.5% | 31.1% |

Paired mean differences were Normal minus Easy `20.684 ± 1.994` and Expert minus Normal `14.052 ± 2.339`, confirming distinct strength on the same scenarios. Average computation per turn was about 0.4 ms for Easy and 9.9 ms for Normal and Expert in this parallel run; these timings exclude UI animation and are machine dependent.

Reproduce that validation with:

```sh
GOMAXPROCS=4 go test -tags botcheck ./internal/game -run TestOpponentCalibration -count=1 -parallel 3 -opponent-games=2000 -opponent-seed=20001 -v
```

Formatting, vet, build, ordinary tests, and race checks passed. The real terminal executable was exercised through a human and opponent turn at each level, including locks, rerolls, a Yahtzee score, and the two-press quit. These were in-progress games; the full-game simulations exercised complete scorecards. The terminal checks do not contribute to player statistics.
