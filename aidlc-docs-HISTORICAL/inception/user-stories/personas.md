# Personas — Squabble

## Persona 1: Alex — The Competitive Enthusiast

- **Background**: Experienced Scrabble player who participates in club play and follows tournament rules closely. Knows the CSW and SOWPODS word lists well and is focused on maximising scores.
- **Goals**: Play challenging games at high AI difficulty; use a recognised tournament dictionary; find the best possible move on every turn.
- **Behaviours**: Sets AI difficulty to 8–10; selects CSW or SOWPODS; studies the board carefully before placing tiles; uses undo sparingly and only when they misclick; saves games they want to analyse later.
- **Pain points**: Gets frustrated when the AI makes weak moves or the dictionary rejects a word they know is valid in their preferred word list.
- **Relevant stories**: US-02 (dictionary selection), US-03 (difficulty), US-07 (tile placement), US-09 (scoring), US-14 (undo), US-15 (save).

---

## Persona 2: Morgan — The Casual Player

- **Background**: Plays Scrabble occasionally with family. Knows the common rules but not tournament edge cases. Prefers a relaxed game against a forgiving AI.
- **Goals**: Enjoy a complete game without the computer crushing them; learn which words are valid without fear of penalty; save a game mid-session and come back later.
- **Behaviours**: Sets AI difficulty to 2–4; uses the default dictionary (OTCWL or "All"); frequently uses undo when they change their mind about a word placement; saves games to finish the next day.
- **Pain points**: Confusion when a word they expect to be valid is rejected; frustration if the game interface is unresponsive or unclear about whose turn it is.
- **Relevant stories**: US-01 (launch), US-02, US-03, US-08 (word validation), US-10 (exchange), US-11 (pass), US-14 (undo), US-15 (save), US-16 (resume).

---

## Persona Mapping Summary

| Story | Alex | Morgan |
|---|---|---|
| US-01 Launch app | ✓ | ✓ |
| US-02 Select dictionary | ✓ | ✓ |
| US-03 Select difficulty | ✓ | ✓ |
| US-04 Start new game | ✓ | ✓ |
| US-05 View initial board | ✓ | ✓ |
| US-06 Draw initial tiles | ✓ | ✓ |
| US-07 Place tiles | ✓ | ✓ |
| US-08 Word validation (no bluff) | ✓ | ✓ |
| US-09 Score a move | ✓ | ✓ |
| US-10 Exchange tiles | | ✓ |
| US-11 Pass a turn | ✓ | ✓ |
| US-12 Watch AI turn | ✓ | ✓ |
| US-13 AI difficulty effect | ✓ | ✓ |
| US-14 Undo last move | ✓ | ✓ |
| US-15 Save game | ✓ | ✓ |
| US-16 Resume saved game | ✓ | ✓ |
| US-17 End: bag empty | ✓ | ✓ |
| US-18 End: six passes | ✓ | ✓ |
| US-19 View final scores | ✓ | ✓ |
