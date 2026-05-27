import React, { useState, useEffect } from "react";

interface Props {
  title: string;
  items: string[];
}

export function App({ title, items }: Props) {
  const [count, setCount] = useState(0);
  const [text, setText] = useState("");

  useEffect(() => {
    document.title = title;
  }, [title]);

  useEffect(() => {
    console.log("count changed", count);
  }, [count]);

  return (
    <div>
      <h1>{title}</h1>
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>+</button>
    </div>
  );
}
