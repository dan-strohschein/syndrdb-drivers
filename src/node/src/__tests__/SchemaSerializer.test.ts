import { SchemaSerializer } from '../schema/SchemaSerializer';
import type { BundleDefinition, SchemaDiff } from '../schema/SchemaDefinition';

describe('SchemaSerializer', () => {
  let serializer: SchemaSerializer;

  beforeEach(() => {
    serializer = new SchemaSerializer();
  });

  describe('serializeDiff', () => {
    it('should serialize empty diff to empty array', () => {
      const diff: SchemaDiff = {
        hasChanges: false,
        bundleChanges: [],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(0);
    });

    it('should serialize bundle creation', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'create',
            bundleName: 'User',
            newDefinition: {
              name: 'User',
              fields: [
                { name: 'id', type: 'STRING', required: true, unique: true },
                { name: 'email', type: 'STRING', required: true, unique: true }
              ],
              indexes: []
            }
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toContain('CREATE BUNDLE "User" WITH FIELDS');
      expect(commands[0]).toContain('id STRING REQUIRED UNIQUE');
      expect(commands[0]).toContain('email STRING REQUIRED UNIQUE');
    });

    it('should serialize bundle deletion', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'delete',
            bundleName: 'Product',
            oldDefinition: {
              name: 'Product',
              fields: [],
              indexes: []
            }
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toBe('DROP BUNDLE "Product"');
    });

    it('should serialize field additions', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'modify',
            bundleName: 'User',
            fieldChanges: [
              {
                type: 'add',
                fieldName: 'name',
                newField: { name: 'name', type: 'STRING', required: false, unique: false }
              }
            ],
            indexChanges: []
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toBe('ALTER BUNDLE "User" ADD FIELD name STRING');
    });

    it('should serialize required field additions', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'modify',
            bundleName: 'User',
            fieldChanges: [
              {
                type: 'add',
                fieldName: 'email',
                newField: { name: 'email', type: 'STRING', required: true, unique: false }
              }
            ],
            indexChanges: []
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands[0]).toBe('ALTER BUNDLE "User" ADD FIELD email STRING REQUIRED');
    });

    it('should serialize unique field additions', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'modify',
            bundleName: 'User',
            fieldChanges: [
              {
                type: 'add',
                fieldName: 'email',
                newField: { name: 'email', type: 'STRING', required: true, unique: true }
              }
            ],
            indexChanges: []
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands[0]).toBe('ALTER BUNDLE "User" ADD FIELD email STRING REQUIRED UNIQUE');
    });

    it('should serialize field removals', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'modify',
            bundleName: 'User',
            fieldChanges: [
              {
                type: 'remove',
                fieldName: 'oldField',
                oldField: { name: 'oldField', type: 'STRING', required: false, unique: false }
              }
            ],
            indexChanges: []
          }
        ],
        indexChanges: []
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toBe('ALTER BUNDLE "User" DROP FIELD oldField');
    });

    it('should serialize hash index creation', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [],
        indexChanges: [
          {
            type: 'create',
            bundleName: 'User',
            index: { fieldName: 'email', type: 'hash' }
          }
        ]
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toBe('CREATE INDEX ON "User" (email) USING HASH');
    });

    it('should serialize btree index creation', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [],
        indexChanges: [
          {
            type: 'create',
            bundleName: 'Product',
            index: { fieldName: 'price', type: 'btree' }
          }
        ]
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toBe('CREATE INDEX ON "Product" (price) USING BTREE');
    });

    it('should serialize index deletion', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [],
        indexChanges: [
          {
            type: 'delete',
            bundleName: 'User',
            index: { fieldName: 'email', type: 'hash' }
          }
        ]
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(1);
      expect(commands[0]).toMatch(/DROP INDEX .+ ON "User"/);
    });

    it('should serialize multiple changes in correct order', () => {
      const diff: SchemaDiff = {
        hasChanges: true,
        bundleChanges: [
          {
            type: 'create',
            bundleName: 'Product',
            newDefinition: {
              name: 'Product',
              fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
              indexes: []
            }
          },
          {
            type: 'modify',
            bundleName: 'User',
            fieldChanges: [
              {
                type: 'add',
                fieldName: 'age',
                newField: { name: 'age', type: 'INT', required: false, unique: false }
              }
            ],
            indexChanges: []
          }
        ],
        indexChanges: [
          {
            type: 'create',
            bundleName: 'User',
            index: { fieldName: 'age', type: 'btree' }
          }
        ]
      };

      const commands = serializer.serializeDiff(diff);

      expect(commands).toHaveLength(3);
      expect(commands[0]).toContain('CREATE BUNDLE "Product"');
      expect(commands[1]).toContain('ALTER BUNDLE "User" ADD FIELD age');
      expect(commands[2]).toContain('CREATE INDEX ON "User" (age) USING BTREE');
    });
  });

  describe('serializeCreateBundle', () => {
    it('should handle bundle with optional fields', () => {
      const bundle: BundleDefinition = {
        name: 'Post',
        fields: [
          { name: 'id', type: 'STRING', required: true, unique: true },
          { name: 'title', type: 'STRING', required: true, unique: false },
          { name: 'content', type: 'TEXT', required: false, unique: false }
        ],
        indexes: []
      };

      const command = serializer.serializeCreateBundle(bundle);

      expect(command).toContain('CREATE BUNDLE "Post" WITH FIELDS');
      expect(command).toContain('id STRING REQUIRED UNIQUE');
      expect(command).toContain('title STRING REQUIRED');
      expect(command).toContain('content TEXT');
      expect(command).not.toContain('content TEXT REQUIRED');
    });

    it('should handle relationship fields', () => {
      const bundle: BundleDefinition = {
        name: 'Order',
        fields: [
          { name: 'id', type: 'STRING', required: true, unique: true },
          { name: 'userId', type: 'relationship', required: true, unique: false, relatedBundle: 'User' }
        ],
        indexes: []
      };

      const command = serializer.serializeCreateBundle(bundle);

      expect(command).toContain('userId RELATIONSHIP(User) REQUIRED');
    });
  });
});
