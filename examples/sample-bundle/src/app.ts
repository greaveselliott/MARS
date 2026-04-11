function add(a: number, b: number): number {
  return a + b;
}

const result: string = add(1, 2); // Type error: number assigned to string
console.log(result);
