# WASM Web Schema Designer

An interactive, browser-based schema designer for SyndrDB. Design your database schema visually and generate TypeScript, JSON Schema, and GraphQL code instantly - all running locally in your browser!

## Features

- 🎨 **Beautiful UI** - Modern, gradient-based design with intuitive layout
- ⚡ **Real-time Generation** - See code output as you type (with smart debouncing)
- 📝 **Multiple Formats** - Generate TypeScript, JSON Schema, and GraphQL simultaneously
- 💾 **Export Options** - Copy to clipboard or download generated files
- 🔌 **Fully Offline** - No server required, all code generation runs in WASM
- 📚 **Example Schema** - Pre-loaded with users/posts example
- 🚀 **Zero Setup** - Just open the HTML file in a browser

## Quick Start

### Option 1: Open Directly

```bash
cd examples/wasm-web
open index.html
```

### Option 2: Local Server (Recommended)

```bash
# Python
python3 -m http.server 8080

# Node.js
npx http-server -p 8080

# Go
go run -m http.server :8080
```

Then open: http://localhost:8080

## Usage

### 1. Load or Edit Schema

- **Use Example**: Click "Load Example" to populate with sample schema
- **Manual Entry**: Edit the JSON schema directly in the editor
- **Paste**: Copy/paste your existing schema

### 2. Generate Code

Code is generated automatically as you type (1 second delay).

Or click "Generate All" to force immediate generation.

### 3. View Output

Switch between tabs to see different formats:
- **JSON Schema** - Draft-07 compatible schemas
- **GraphQL** - SDL type definitions with queries
- **TypeScript** - Typed interfaces

### 4. Export Code

- **Copy** - Click copy button to copy code to clipboard
- **Download** - Click download button to save as file

## Schema Format

The schema editor accepts JSON with this structure:

```json
{
  "bundles": [
    {
      "name": "users",
      "fields": [
        {
          "name": "id",
          "type": "int",
          "required": true,
          "unique": true
        },
        {
          "name": "email",
          "type": "string",
          "required": true,
          "unique": true
        }
      ],
      "indexes": [
        {
          "name": "idx_email",
          "type": "hash",
          "fields": ["email"]
        }
      ],
      "relationships": []
    }
  ]
}
```

### Field Properties

- `name` - Field name (string)
- `type` - Data type: `int`, `float`, `string`, `bool`, `timestamp`, `json`
- `required` - Is field required? (boolean)
- `unique` - Must values be unique? (boolean)

### Index Properties

- `name` - Index name (string)
- `type` - Index type: `hash`, `btree`, `fulltext`
- `fields` - Array of field names to index

## Generated Code Examples

### TypeScript Output

```typescript
export interface User {
  id: number;
  email: string;
  username: string;
  created_at: Date;
  updated_at?: Date;
}

export interface Post {
  id: number;
  user_id: number;
  title: string;
  content: string;
  published: boolean;
}
```

### JSON Schema Output

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "users": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "email", "username"],
        "properties": {
          "id": { "type": "integer" },
          "email": { "type": "string" },
          "username": { "type": "string" }
        }
      }
    }
  }
}
```

### GraphQL Output

```graphql
type User {
  id: Int!
  email: String!
  username: String!
  created_at: String!
  updated_at: String
}

type Query {
  users: [User!]!
  user(id: Int!): User
}
```

## Features in Detail

### Auto-Generation

- Generates code automatically after 1 second of no typing
- Prevents excessive generation while editing
- Updates all three formats simultaneously

### Status Indicator

- **Animated dots** - WASM driver is initializing
- **Green checkmark** - Driver ready, showing version
- **Red X** - Error loading driver

### Error Handling

- Invalid JSON shows error message
- Empty schema shows helpful placeholder
- WASM errors displayed in output

### Browser Compatibility

- ✅ Chrome/Edge (recommended)
- ✅ Firefox
- ✅ Safari
- ⚠️ IE not supported (no WASM)

## Use Cases

### 1. Schema Design

Design your database schema visually before implementing:

```
1. Start with example or blank schema
2. Add/modify bundles and fields
3. See generated types in real-time
4. Download TypeScript for frontend
5. Download GraphQL for API
```

### 2. Documentation

Generate documentation for existing schemas:

```
1. Paste your existing schema.json
2. Generate GraphQL SDL for API docs
3. Download and commit to repo
```

### 3. Rapid Prototyping

Quickly prototype database structures:

```
1. Design schema in browser
2. Download JSON schema
3. Use with migration-cli to create database
4. Download TypeScript types for app
```

### 4. Learning

Explore SyndrDB schema design:

```
1. Modify example schema
2. See how changes affect generated code
3. Understand type mappings
4. Experiment with indexes and relationships
```

## Keyboard Shortcuts

- **Ctrl/Cmd + S** - Download current tab content
- **Tab in editor** - Insert 2 spaces (not tab character)
- **Ctrl/Cmd + Z** - Undo in editor
- **Ctrl/Cmd + Shift + Z** - Redo in editor

## Technical Details

### WASM Driver

- **Size**: 4.2MB uncompressed (1.1MB gzipped)
- **Load Time**: ~100-300ms on modern hardware
- **Performance**: Sub-millisecond code generation
- **Memory**: ~10-20MB baseline

### Code Generation

All code generation happens locally:

- **JSON Schema** - Generated by WASM driver (`SyndrDB.generateJSONSchema()`)
- **GraphQL** - Generated by WASM driver (`SyndrDB.generateGraphQLSchema()`)
- **TypeScript** - Generated by JavaScript (client-side implementation)

### Why Not Database Connection?

Browsers cannot make raw TCP connections, so this demo focuses on **schema design and code generation** - things that work perfectly offline.

For database operations, see `../node-graphql-api/` or use the Go driver directly.

## Deployment

### Static Hosting

Deploy to any static host:

**GitHub Pages:**

```bash
# Copy files to gh-pages branch
cp index.html ../../wasm/syndrdb.wasm ../../wasm/wasm_exec.js /path/to/gh-pages/
```

**Netlify:**

```bash
# Deploy directory
netlify deploy --dir=. --prod
```

**Vercel:**

```bash
vercel --prod
```

### CDN

Files needed:
- `index.html` (360KB)
- `syndrdb.wasm` (4.2MB)
- `wasm_exec.js` (14KB)

Serve with:
- `Content-Type: application/wasm` for `.wasm` files
- `Content-Type: application/javascript` for `.js` files
- HTTPS recommended for Service Workers

### Embedded

Embed in your own site:

```html
<iframe src="https://yoursite.com/schema-designer/" width="100%" height="800px"></iframe>
```

## Limitations

- **No database connection** - Browser TCP limitations
- **Client-side only** - Cannot save to server
- **No collaboration** - Single-user experience
- **Basic validation** - JSON structure only

For full functionality, use the Go CLI tools or Node.js examples.

## Troubleshooting

### WASM Won't Load

- **Check file paths** - Ensure `syndrdb.wasm` and `wasm_exec.js` are in `../../wasm/`
- **Use HTTP server** - Don't open `file://` directly (CORS issues)
- **Check browser console** - Look for error messages

### Generation Not Working

- **Validate JSON** - Ensure schema is valid JSON
- **Check format** - Must have `bundles` array
- **Wait for init** - Green checkmark means ready

### Copy/Download Fails

- **HTTPS required** - Clipboard API needs secure context
- **Check permissions** - Browser may block clipboard access
- **Try download** - Alternative to clipboard

## Browser Console

For debugging, these are available globally:

```javascript
// Check driver status
console.log(SyndrDB);

// Manual generation
const jsonSchema = await SyndrDB.generateJSONSchema(schemaText, 'single');
const graphql = await SyndrDB.generateGraphQLSchema(schemaText);

// Check version
console.log(SyndrDB.version);
```

## What This Demonstrates

- ✅ WASM driver in browser environment
- ✅ Interactive schema design
- ✅ Real-time code generation
- ✅ Multi-format output
- ✅ Modern UI/UX patterns
- ✅ Offline-first approach

## Future Enhancements

Potential additions:
- Visual schema editor (drag-and-drop)
- Schema validation and hints
- Import from database
- Export to migration files
- Collaboration features
- Schema version history
- Dark mode theme
- Custom color themes

## Related Examples

- `../schema-codegen/` - CLI tool for code generation
- `../migration-cli/` - Create and apply migrations
- `../node-graphql-api/` - Use generated types in API
- `../node-migration/` - Automated schema deployment

## Support

If you encounter issues:

1. Check browser console for errors
2. Verify WASM files are present
3. Try different browser
4. Open GitHub issue with details

## License

Same as SyndrDB Go driver - see LICENSE file.
