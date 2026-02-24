/**
 * Lightweight sequential-character fuzzy matcher.
 * Scores based on consecutive matches, start-of-word bonus, and prefix bonus.
 */
export function fuzzyMatch(query: string, text: string): { matches: boolean; score: number } {
  const lowerQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();

  // Exact substring match gets a big bonus
  if (lowerText.includes(lowerQuery)) {
    const index = lowerText.indexOf(lowerQuery);
    // Prefix match is best
    const prefixBonus = index === 0 ? 100 : 0;
    // Start-of-word bonus
    const wordBonus = index > 0 && /\W/.test(text[index - 1]) ? 50 : 0;
    return { matches: true, score: 200 + prefixBonus + wordBonus + lowerQuery.length };
  }

  // Sequential character matching
  let qi = 0;
  let score = 0;
  let consecutive = 0;

  for (let ti = 0; ti < lowerText.length && qi < lowerQuery.length; ti++) {
    if (lowerText[ti] === lowerQuery[qi]) {
      qi++;
      consecutive++;
      // Consecutive match bonus grows quadratically
      score += consecutive * 2;
      // Start-of-word bonus
      if (ti === 0 || /\W/.test(text[ti - 1])) {
        score += 10;
      }
    } else {
      consecutive = 0;
    }
  }

  if (qi < lowerQuery.length) {
    return { matches: false, score: 0 };
  }

  return { matches: true, score };
}
