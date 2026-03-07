/**
 * Shared formatting utilities for the accounting tool.
 * DB stores expenses as negative amounts. These helpers display them as positive
 * (the natural way to think about spending), with refunds marked as "+".
 */

/** Format a DB amount for expense-centric display.
 *  Negative (expense) → £123.45
 *  Positive (refund)  → +£10.00
 */
export function formatExpense(amount: number, symbol = '£'): string {
  if (amount > 0) {
    return `+${symbol}${amount.toFixed(2)}`;
  }
  return `${symbol}${Math.abs(amount).toFixed(2)}`;
}

/** Format a date string to "02 Mar 2026" style. */
export function formatDate(dateStr: string | null | undefined, locale = 'en-GB'): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleDateString(locale, {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
}

/** Format currency with sign preserved (for contexts where sign matters). */
export function formatCurrency(value: number, symbol = '£'): string {
  const sign = value < 0 ? '-' : '';
  return `${sign}${symbol}${Math.abs(value).toFixed(2)}`;
}
