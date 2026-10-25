export async function copyText(text: string): Promise<void> {
  if (!text) throw new Error('There is no text to copy.');
  if (!navigator.clipboard?.writeText) throw new Error('Clipboard access is unavailable.');
  await navigator.clipboard.writeText(text);
}
