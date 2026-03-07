import { useEffect, useState, useCallback } from 'react';

interface ToastMessage {
  id: number;
  text: string;
  type: 'error' | 'success' | 'info';
}

let toastId = 0;
let addToastFn: ((msg: Omit<ToastMessage, 'id'>) => void) | null = null;

/** Show a toast notification from anywhere. */
export function toast(text: string, type: 'error' | 'success' | 'info' = 'info') {
  addToastFn?.({ text, type });
}

export default function ToastContainer() {
  const [messages, setMessages] = useState<ToastMessage[]>([]);

  const addToast = useCallback((msg: Omit<ToastMessage, 'id'>) => {
    const id = ++toastId;
    setMessages((prev) => [...prev, { ...msg, id }]);
    setTimeout(() => {
      setMessages((prev) => prev.filter((m) => m.id !== id));
    }, 4000);
  }, []);

  useEffect(() => {
    addToastFn = addToast;
    return () => { addToastFn = null; };
  }, [addToast]);

  if (messages.length === 0) return null;

  return (
    <div className="toast-container">
      {messages.map((msg) => (
        <div key={msg.id} className={`toast toast-${msg.type}`}>
          {msg.text}
        </div>
      ))}
    </div>
  );
}
