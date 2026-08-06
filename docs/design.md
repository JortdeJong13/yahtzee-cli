# Yahtzee CLI design

## Scope

The first version is a local, terminal-only game of Yahtzee for one human player against an expected-value opponent. It starts directly in the game and has no welcome screen, menu, saved state, network mode, or difficulty selector.

## Turn flow

1. A new game creates five random dice. They are displayed but do not count as a roll.
2. The human player presses `r` to make the first roll.
3. A turn allows at most three rolls. Unlocked dice are rerolled; locked dice are kept.
   Each roll briefly cycles the faces of unlocked dice before showing the settled result.
4. The human can toggle dice with `1` through `5`, move through open score categories with the arrow keys, and confirm a category with Enter.
5. After the human scores, the opponent takes its turn automatically. The opponent evaluates legal hold choices by expected value and selects a score using category opportunity costs.
6. Dice values carry into the next turn. Lock state, roll count, and the rolled flag reset; the next first roll rerolls all unlocked dice.
7. After both players have filled all thirteen categories, the final board and winner remain as regular terminal output.

## Rules

The game uses the classic Yahtzee scorecard:

- Ones through Sixes: the sum of matching dice.
- Three of a Kind: total of all dice when at least three match.
- Four of a Kind: total of all dice when at least four match.
- Full House: 25 points for three of one value and two of another.
- Small Straight: 30 points for at least four consecutive values.
- Large Straight: 40 points for five consecutive values.
- Yahtzee: 50 points for five matching dice.
- Chance: total of all dice.
- Upper bonus: 35 points when the upper section reaches 63.

The extra Yahtzee bonus and Joker behavior are implemented as the standard classic rules: a later Yahtzee after scoring 50 in the Yahtzee box earns 100 bonus points and follows the corresponding upper-section/lower-section Joker rules.

## Terminal behavior

The live board is rendered on an ANSI alternate screen so it can be redrawn without leaving stale frames behind. If an in-progress game is quit, its board is discarded. When a game ends, the final board without live controls is immediately copied to the normal terminal buffer; the quit/restart prompt is then shown on a fresh alternate screen and is not copied. Restarting therefore leaves previous completed-game results in scrollback.

The game uses a compact two-column layout. Five dice in a 5-face arrangement sit above the current turn status on the left. The right column contains both scorecards inside a bordered panel. The currently selected score row is bold. All controls share one row below the dice and scorecard.

Colors have a single purpose each:

- Green: human turn and achieved upper bonus.
- Yellow: opponent turn.
- Cyan: locked dice.
- Bright accent: available scores, the selected score arrow, and the text after the human turn prefix.
- Dim gray: filled categories and unavailable actions.

Pressing `q` once arms quit confirmation. Pressing `q` again quits. Any other key cancels the confirmation. A terminal does not reliably expose key-release events, so a literal hold gesture is not used.

## Deliberate non-goals

- Difficulty levels.
- Sound.
- Game persistence or replay files.
- Multiplayer or network play.
- A general-purpose Yahtzee engine API.
