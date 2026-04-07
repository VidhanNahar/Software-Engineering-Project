/**
 * Formats a number as a currency string.
 * Standardized for the Indian market (INR / ₹).
 */
export const formatCurrency = (
  value: number | undefined | null
): string => {
  if (value === undefined || value === null) return '—';
  
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
};

/**
 * Formats a number as a currency string (₹).
 */
export const formatPrice = (value: number | undefined | null): string => {
  return formatCurrency(value);
};
