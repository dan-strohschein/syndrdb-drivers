import http from 'http';
import fs from 'fs';
import { graphql, buildSchema } from 'graphql';

// Load WASM driver
const go = new Go();
const wasmBinary = fs.readFileSync('../../wasm/syndrdb.wasm');
const result = await WebAssembly.instantiate(wasmBinary, go.importObject);
go.run(result.instance);

// Wait for WASM to initialize
await new Promise(resolve => setTimeout(resolve, 100));

const PORT = process.env.PORT || 3000;
const SYNDR_CONN = process.env.SYNDR_CONN || 'syndrdb://127.0.0.1:1776:primary:root:root;';

// Initialize SyndrDB client
console.log('Initializing SyndrDB client...');
await SyndrDB.createClient({
  defaultTimeoutMs: 10000,
  debugMode: false,
  maxRetries: 3
});

// Connect to database
console.log('Connecting to SyndrDB...');
try {
  await SyndrDB.connect(SYNDR_CONN);
  console.log('✓ Connected to SyndrDB');
} catch (err) {
  console.error('Failed to connect:', err);
  process.exit(1);
}

// GraphQL Schema - matches a typical todo app
const schema = buildSchema(`
  type Todo {
    id: Int!
    title: String!
    description: String
    completed: Boolean!
    created_at: String!
    updated_at: String
  }

  input CreateTodoInput {
    title: String!
    description: String
    completed: Boolean
  }

  input UpdateTodoInput {
    title: String
    description: String
    completed: Boolean
  }

  type Query {
    todos: [Todo!]!
    todo(id: Int!): Todo
    completedTodos: [Todo!]!
    pendingTodos: [Todo!]!
  }

  type Mutation {
    createTodo(input: CreateTodoInput!): Todo!
    updateTodo(id: Int!, input: UpdateTodoInput!): Todo!
    deleteTodo(id: Int!): Boolean!
    toggleTodo(id: Int!): Todo!
  }
`);

// Resolvers
const root = {
  // Queries
  todos: async () => {
    const result = await SyndrDB.query('SELECT * FROM todos ORDER BY created_at DESC;', 10000);
    return parseTodos(result);
  },

  todo: async ({ id }) => {
    const result = await SyndrDB.query(`SELECT * FROM todos WHERE id = ${id};`, 10000);
    const todos = parseTodos(result);
    return todos[0] || null;
  },

  completedTodos: async () => {
    const result = await SyndrDB.query('SELECT * FROM todos WHERE completed = true ORDER BY created_at DESC;', 10000);
    return parseTodos(result);
  },

  pendingTodos: async () => {
    const result = await SyndrDB.query('SELECT * FROM todos WHERE completed = false ORDER BY created_at DESC;', 10000);
    return parseTodos(result);
  },

  // Mutations
  createTodo: async ({ input }) => {
    const now = new Date().toISOString();
    const completed = input.completed || false;
    const description = input.description || null;
    
    // Get next ID
    const countResult = await SyndrDB.query('SELECT COUNT(*) as count FROM todos;', 10000);
    const nextId = (countResult?.count || 0) + 1;

    const mutation = `
      INSERT INTO todos (id, title, description, completed, created_at) 
      VALUES (${nextId}, "${escapeString(input.title)}", ${description ? `"${escapeString(description)}"` : 'null'}, ${completed}, "${now}");
    `;
    
    await SyndrDB.mutate(mutation, 10000);
    
    return {
      id: nextId,
      title: input.title,
      description: description,
      completed: completed,
      created_at: now,
      updated_at: null
    };
  },

  updateTodo: async ({ id, input }) => {
    const now = new Date().toISOString();
    const updates = [];
    
    if (input.title !== undefined) {
      updates.push(`title = "${escapeString(input.title)}"`);
    }
    if (input.description !== undefined) {
      updates.push(`description = ${input.description ? `"${escapeString(input.description)}"` : 'null'}`);
    }
    if (input.completed !== undefined) {
      updates.push(`completed = ${input.completed}`);
    }
    updates.push(`updated_at = "${now}"`);

    const mutation = `UPDATE todos SET ${updates.join(', ')} WHERE id = ${id};`;
    await SyndrDB.mutate(mutation, 10000);

    // Fetch updated todo
    const result = await SyndrDB.query(`SELECT * FROM todos WHERE id = ${id};`, 10000);
    const todos = parseTodos(result);
    return todos[0];
  },

  deleteTodo: async ({ id }) => {
    const mutation = `DELETE FROM todos WHERE id = ${id};`;
    await SyndrDB.mutate(mutation, 10000);
    return true;
  },

  toggleTodo: async ({ id }) => {
    // First get current state
    const result = await SyndrDB.query(`SELECT completed FROM todos WHERE id = ${id};`, 10000);
    const todos = parseTodos(result);
    if (!todos[0]) {
      throw new Error('Todo not found');
    }

    const newCompleted = !todos[0].completed;
    const now = new Date().toISOString();
    
    const mutation = `UPDATE todos SET completed = ${newCompleted}, updated_at = "${now}" WHERE id = ${id};`;
    await SyndrDB.mutate(mutation, 10000);

    // Return updated todo
    const updated = await SyndrDB.query(`SELECT * FROM todos WHERE id = ${id};`, 10000);
    return parseTodos(updated)[0];
  }
};

// Helper functions
function parseTodos(result) {
  if (!result || !Array.isArray(result)) {
    return [];
  }
  return result.map(row => ({
    id: row.id,
    title: row.title,
    description: row.description,
    completed: row.completed,
    created_at: row.created_at,
    updated_at: row.updated_at
  }));
}

function escapeString(str) {
  return str.replace(/"/g, '\\"').replace(/\n/g, '\\n');
}

// HTTP Server
const server = http.createServer(async (req, res) => {
  // Enable CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  if (req.url === '/health') {
    const state = SyndrDB.getState();
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      status: 'ok', 
      database: state,
      version: SyndrDB.getVersion()
    }));
    return;
  }

  if (req.url === '/graphql' && req.method === 'POST') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      try {
        const { query, variables } = JSON.parse(body);
        
        const result = await graphql({
          schema,
          source: query,
          rootValue: root,
          variableValues: variables
        });

        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(result));
      } catch (err) {
        console.error('GraphQL Error:', err);
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ 
          errors: [{ message: err.message }] 
        }));
      }
    });
    return;
  }

  if (req.url === '/' && req.method === 'GET') {
    // Serve GraphQL Playground
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(GRAPHQL_PLAYGROUND_HTML);
    return;
  }

  res.writeHead(404);
  res.end('Not Found');
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Shutting down gracefully...');
  await SyndrDB.disconnect();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', async () => {
  console.log('\nShutting down gracefully...');
  await SyndrDB.disconnect();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

server.listen(PORT, () => {
  console.log(`\n🚀 GraphQL API running at http://localhost:${PORT}/graphql`);
  console.log(`📊 GraphQL Playground at http://localhost:${PORT}/`);
  console.log(`❤️  Health check at http://localhost:${PORT}/health`);
});

// GraphQL Playground HTML
const GRAPHQL_PLAYGROUND_HTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>SyndrDB GraphQL API</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f5f5f5; }
    .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
    h1 { color: #333; margin-bottom: 10px; }
    .subtitle { color: #666; margin-bottom: 30px; }
    .playground { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    textarea { width: 100%; height: 300px; padding: 15px; border: 1px solid #ddd; border-radius: 4px; font-family: 'Monaco', monospace; font-size: 14px; resize: vertical; }
    button { background: #0070f3; color: white; border: none; padding: 12px 24px; border-radius: 4px; font-size: 14px; cursor: pointer; margin-top: 10px; }
    button:hover { background: #0051cc; }
    .response { margin-top: 20px; }
    pre { background: #f8f8f8; padding: 15px; border-radius: 4px; overflow-x: auto; border: 1px solid #ddd; }
    .examples { margin-top: 30px; }
    .example { background: #f8f8f8; padding: 15px; margin-bottom: 15px; border-radius: 4px; border-left: 4px solid #0070f3; cursor: pointer; }
    .example:hover { background: #f0f0f0; }
    .example h3 { color: #333; margin-bottom: 5px; font-size: 16px; }
    .example p { color: #666; font-size: 14px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>🚀 SyndrDB GraphQL API</h1>
    <p class="subtitle">GraphQL interface powered by SyndrDB WASM driver</p>
    
    <div class="playground">
      <h2>Execute Query</h2>
      <textarea id="query" placeholder="Enter your GraphQL query...">query {
  todos {
    id
    title
    completed
  }
}</textarea>
      <button onclick="executeQuery()">Execute</button>
      
      <div class="response">
        <h3>Response:</h3>
        <pre id="response">Results will appear here...</pre>
      </div>
    </div>

    <div class="examples">
      <h2>Example Queries</h2>
      
      <div class="example" onclick="setQuery(examples.allTodos)">
        <h3>Get All Todos</h3>
        <p>Fetch all todos from the database</p>
      </div>
      
      <div class="example" onclick="setQuery(examples.createTodo)">
        <h3>Create Todo</h3>
        <p>Add a new todo item</p>
      </div>
      
      <div class="example" onclick="setQuery(examples.updateTodo)">
        <h3>Update Todo</h3>
        <p>Update an existing todo</p>
      </div>
      
      <div class="example" onclick="setQuery(examples.toggleTodo)">
        <h3>Toggle Todo</h3>
        <p>Toggle completion status</p>
      </div>
      
      <div class="example" onclick="setQuery(examples.deleteTodo)">
        <h3>Delete Todo</h3>
        <p>Remove a todo from the database</p>
      </div>
    </div>
  </div>

  <script>
    const examples = {
      allTodos: \`query {
  todos {
    id
    title
    description
    completed
    created_at
  }
}\`,
      createTodo: \`mutation {
  createTodo(input: {
    title: "Learn SyndrDB"
    description: "Explore GraphQL API features"
    completed: false
  }) {
    id
    title
    completed
    created_at
  }
}\`,
      updateTodo: \`mutation {
  updateTodo(id: 1, input: {
    title: "Updated title"
    completed: true
  }) {
    id
    title
    completed
    updated_at
  }
}\`,
      toggleTodo: \`mutation {
  toggleTodo(id: 1) {
    id
    completed
    updated_at
  }
}\`,
      deleteTodo: \`mutation {
  deleteTodo(id: 1)
}\`
    };

    function setQuery(query) {
      document.getElementById('query').value = query;
    }

    async function executeQuery() {
      const query = document.getElementById('query').value;
      const responseEl = document.getElementById('response');
      
      responseEl.textContent = 'Executing...';
      
      try {
        const response = await fetch('/graphql', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query })
        });
        
        const result = await response.json();
        responseEl.textContent = JSON.stringify(result, null, 2);
      } catch (err) {
        responseEl.textContent = 'Error: ' + err.message;
      }
    }
  </script>
</body>
</html>
`;
