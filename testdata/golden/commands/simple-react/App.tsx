import React, { useState, useEffect } from 'react';

export function App() {
  const [count, setCount] = useState(0);
  const [text, setText] = useState('');
  const [items, setItems] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  useEffect(() => {
    document.title = `Count: ${count}`;
  }, [count]);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/items?q=${search}`)
      .then((r) => r.json())
      .then((data) => {
        setItems(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [search]);

  useEffect(() => {
    console.log('items updated:', items.length);
  }, [items]);

  return (
    <div>
      <h1>App</h1>
      <input
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Search..."
      />
      <p>Count: {count}</p>
      <button onClick={() => setCount((c) => c + 1)}>+</button>
      {loading && <p>Loading...</p>}
      <ul>
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}
