/**
 * Unit tests for SchemaManager
 */

import { SchemaManager, SchemaBuilder } from '../../src/features/schema-manager';
import { ValidationError } from '../../src/errors';
import type { SyndrDBClient } from '../../src/client';
import type { SchemaDefinition, BundleDefinition } from '../../src/types/schema';

describe('SchemaManager', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let manager: SchemaManager;

  beforeEach(() => {
    mockClient = {
      generateJSONSchema: jest.fn().mockResolvedValue('{"type": "object"}'),
      generateGraphQLSchema: jest.fn().mockResolvedValue('type Query { hello: String }'),
    } as any;

    manager = new SchemaManager(mockClient);
  });

  describe('generateJSONSchema', () => {
    it('should generate JSON schema in single mode', async () => {
      const schema = await manager.generateJSONSchema('single');

      expect(mockClient.generateJSONSchema).toHaveBeenCalledWith('single');
      expect(schema).toBe('{"type": "object"}');
    });

    it('should generate JSON schema in multiple mode', async () => {
      const schema = await manager.generateJSONSchema('multiple');

      expect(mockClient.generateJSONSchema).toHaveBeenCalledWith('multiple');
    });

    it('should throw on error', async () => {
      mockClient.generateJSONSchema.mockRejectedValueOnce(new Error('Generation failed'));

      await expect(manager.generateJSONSchema()).rejects.toThrow(ValidationError);
    });
  });

  describe('generateGraphQLSchema', () => {
    it('should generate GraphQL schema', async () => {
      const schema = await manager.generateGraphQLSchema({
        includeMutations: true,
      });

      expect(mockClient.generateGraphQLSchema).toHaveBeenCalled();
      expect(schema).toContain('type Query');
    });

    it('should throw on error', async () => {
      mockClient.generateGraphQLSchema.mockRejectedValueOnce(new Error('Generation failed'));

      await expect(manager.generateGraphQLSchema()).rejects.toThrow(ValidationError);
    });
  });

  describe('generateTypeScript', () => {
    const schema: SchemaDefinition = {
      bundles: [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'INT', required: true, unique: true },
            { name: 'name', type: 'STRING', required: true, unique: false },
            { name: 'email', type: 'STRING', required: false, unique: true },
          ],
        },
      ],
    };

    it('should generate TypeScript interfaces', () => {
      const ts = manager.generateTypeScript(schema);

      expect(ts).toContain('export interface User');
      expect(ts).toContain('id: number;');
      expect(ts).toContain('name: string;');
      expect(ts).toContain('email?: string;');
    });

    it('should include JSDoc comments', () => {
      const schemaWithDocs: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [{ name: 'id', type: 'INT', required: true, unique: false }],
            metadata: { description: 'User entity' },
          },
        ],
      };

      const ts = manager.generateTypeScript(schemaWithDocs, {
        includeJSDoc: true,
      });

      expect(ts).toContain('/**');
      expect(ts).toContain('User entity');
    });

    it('should generate type guards', () => {
      const ts = manager.generateTypeScript(schema, {
        generateTypeGuards: true,
      });

      expect(ts).toContain('export function isUser');
      expect(ts).toContain('obj is User');
    });

    it('should use custom type mappings', () => {
      const ts = manager.generateTypeScript(schema, {
        typeMappings: {
          INT: 'bigint',
          STRING: 'string',
        },
      });

      expect(ts).toContain('id: bigint;');
      expect(ts).toContain('name: string;');
    });

    it('should handle relationship fields', () => {
      const schemaWithRels: SchemaDefinition = {
        bundles: [
          {
            name: 'Post',
            fields: [
              { name: 'id', type: 'INT', required: true, unique: true },
              {
                name: 'author',
                type: 'relationship',
                required: true,
                unique: false,
                relatedBundle: 'User',
              },
            ],
          },
        ],
      };

      const ts = manager.generateTypeScript(schemaWithRels);

      expect(ts).toContain('author: User;');
    });
  });

  describe('validateSchema', () => {
    it('should return no errors for valid schema', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [
              { name: 'id', type: 'INT', required: true, unique: true },
              { name: 'name', type: 'STRING', required: true, unique: false },
            ],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors).toHaveLength(0);
    });

    it('should error on empty bundles', () => {
      const schema: SchemaDefinition = {
        bundles: [],
      };

      const errors = manager.validateSchema(schema);
      expect(errors).toContain('Schema must contain at least one bundle');
    });

    it('should error on duplicate bundle names', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [{ name: 'id', type: 'INT', required: true, unique: true }],
          },
          {
            name: 'User',
            fields: [{ name: 'id', type: 'INT', required: true, unique: true }],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('Duplicate bundle name'))).toBe(true);
    });

    it('should error on bundle without fields', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'Empty',
            fields: [],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('must contain at least one field'))).toBe(true);
    });

    it('should error on duplicate field names', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [
              { name: 'id', type: 'INT', required: true, unique: true },
              { name: 'id', type: 'STRING', required: true, unique: false },
            ],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('Duplicate field name'))).toBe(true);
    });

    it('should error on relationship without relatedBundle', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'Post',
            fields: [
              { name: 'author', type: 'relationship', required: true, unique: false },
            ],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('must specify relatedBundle'))).toBe(true);
    });

    it('should error on relationship to unknown bundle', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'Post',
            fields: [
              {
                name: 'author',
                type: 'relationship',
                required: true,
                unique: false,
                relatedBundle: 'User',
              },
            ],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('references unknown bundle'))).toBe(true);
    });

    it('should allow self-referencing relationships', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [
              { name: 'id', type: 'INT', required: true, unique: true },
              {
                name: 'manager',
                type: 'relationship',
                required: false,
                unique: false,
                relatedBundle: 'User',
              },
            ],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors).toHaveLength(0);
    });

    it('should error on index referencing unknown field', () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'User',
            fields: [{ name: 'id', type: 'INT', required: true, unique: true }],
            indexes: [{ fieldName: 'email', type: 'hash' }],
          },
        ],
      };

      const errors = manager.validateSchema(schema);
      expect(errors.some((e) => e.includes('references unknown field'))).toBe(true);
    });
  });

  describe('SchemaBuilder', () => {
    it('should create empty schema', () => {
      const builder = SchemaManager.builder();
      const schema = builder.build();

      expect(schema.bundles).toHaveLength(0);
    });

    it('should build schema with bundles', () => {
      const schema = SchemaManager.builder()
        .bundle('User')
        .field({ name: 'id', type: 'INT', required: true, unique: true })
        .field({ name: 'name', type: 'STRING', required: true, unique: false })
        .bundle('Post')
        .field({ name: 'id', type: 'INT', required: true, unique: true })
        .build();

      expect(schema.bundles).toHaveLength(2);
      expect(schema.bundles[0].name).toBe('User');
      expect(schema.bundles[0].fields).toHaveLength(2);
      expect(schema.bundles[1].name).toBe('Post');
    });

    it('should add indexes', () => {
      const schema = SchemaManager.builder()
        .bundle('User')
        .field({ name: 'email', type: 'STRING', required: true, unique: false })
        .index({ fieldName: 'email', type: 'hash' })
        .build();

      expect(schema.bundles[0].indexes).toHaveLength(1);
      expect(schema.bundles[0].indexes![0].fieldName).toBe('email');
    });

    it('should set version', () => {
      const schema = SchemaManager.builder().version(2).build();

      expect(schema.version).toBe(2);
    });

    it('should set metadata', () => {
      const schema = SchemaManager.builder()
        .metadata({ description: 'My schema' })
        .build();

      expect(schema.metadata?.description).toBe('My schema');
    });

    it('should throw when adding field without bundle', () => {
      const builder = SchemaManager.builder();

      expect(() =>
        builder.field({ name: 'id', type: 'INT', required: true, unique: true })
      ).toThrow('Must call bundle()');
    });
  });
});
