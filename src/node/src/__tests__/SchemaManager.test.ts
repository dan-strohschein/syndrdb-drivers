import { SchemaManager } from '../schema/SchemaManager';
import type { BundleDefinition } from '../schema/SchemaDefinition';

describe('SchemaManager', () => {
  let schemaManager: SchemaManager;

  beforeEach(() => {
    schemaManager = new SchemaManager();
  });

  describe('parseServerSchema', () => {
    it('should parse valid server response with bundles', () => {
      const serverResponse = {
        ExecutionTimeMS: 5,
        Result: [
          {
            BundleName: 'User',
            BundleMetadata: {
              DocumentStructure: {
                FieldDefinitions: {
                  id: { Type: 'STRING', Required: true, Unique: true },
                  email: { Type: 'STRING', Required: true, Unique: true },
                  name: { Type: 'STRING', Required: true, Unique: false }
                },
                Indexes: {
                  email_idx: {
                    IndexType: 'hash',
                    HashIndexField: 'email'
                  }
                }
              }
            }
          }
        ],
        ResultCount: 1
      };

      const bundles = schemaManager.parseServerSchema(serverResponse);

      expect(bundles).toHaveLength(1);
      expect(bundles[0].name).toBe('User');
      expect(bundles[0].fields).toHaveLength(3);
      expect(bundles[0].indexes).toHaveLength(1);
      expect(bundles[0].fields[0].name).toBe('id');
      expect(bundles[0].fields[0].type).toBe('STRING');
      expect(bundles[0].fields[0].required).toBe(true);
      expect(bundles[0].fields[0].unique).toBe(true);
      expect(bundles[0].indexes[0].fieldName).toBe('email');
      expect(bundles[0].indexes[0].type).toBe('hash');
    });

    it('should handle empty server response', () => {
      const serverResponse = {
        ExecutionTimeMS: 1,
        Result: [],
        ResultCount: 0
      };

      const bundles = schemaManager.parseServerSchema(serverResponse);

      expect(bundles).toHaveLength(0);
    });

    it('should parse BTree indexes correctly', () => {
      const serverResponse = {
        ExecutionTimeMS: 3,
        Result: [
          {
            BundleName: 'Product',
            BundleMetadata: {
              DocumentStructure: {
                FieldDefinitions: {
                  id: { Type: 'STRING', Required: true, Unique: true },
                  price: { Type: 'FLOAT', Required: true, Unique: false }
                },
                Indexes: {
                  price_idx: {
                    IndexType: 'btree',
                    BTreeIndexField: 'price'
                  }
                }
              }
            }
          }
        ],
        ResultCount: 1
      };

      const bundles = schemaManager.parseServerSchema(serverResponse);

      expect(bundles[0].indexes[0].type).toBe('btree');
      expect(bundles[0].indexes[0].fieldName).toBe('price');
    });

    it('should handle relationship fields', () => {
      const serverResponse = {
        ExecutionTimeMS: 4,
        Result: [
          {
            BundleName: 'Order',
            BundleMetadata: {
              DocumentStructure: {
                FieldDefinitions: {
                  id: { Type: 'STRING', Required: true, Unique: true },
                  userId: { Type: 'relationship', Required: true, Unique: false, RelatedBundle: 'User' }
                },
                Indexes: {}
              }
            }
          }
        ],
        ResultCount: 1
      };

      const bundles = schemaManager.parseServerSchema(serverResponse);

      expect(bundles[0].fields[1].type).toBe('relationship');
      expect(bundles[0].fields[1].relatedBundle).toBe('User');
    });
  });

  describe('compareSchemas', () => {
    it('should detect no changes when schemas match', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: [{ fieldName: 'email', type: 'hash' }]
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: [{ fieldName: 'email', type: 'hash' }]
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(false);
      expect(diff.bundleChanges).toHaveLength(0);
      expect(diff.indexChanges).toHaveLength(0);
    });

    it('should detect new bundle creation', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        },
        {
          name: 'Product',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.bundleChanges).toHaveLength(1);
      expect(diff.bundleChanges[0].type).toBe('create');
      expect(diff.bundleChanges[0].bundleName).toBe('Product');
    });

    it('should detect bundle deletion', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        },
        {
          name: 'Product',
          fields: [{ name: 'id', type: 'STRING', required: true, unique: true }],
          indexes: []
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.bundleChanges).toHaveLength(1);
      expect(diff.bundleChanges[0].type).toBe('delete');
      expect(diff.bundleChanges[0].bundleName).toBe('Product');
    });

    it('should detect field additions', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true },
            { name: 'name', type: 'STRING', required: false, unique: false }
          ],
          indexes: []
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.bundleChanges).toHaveLength(1);
      expect(diff.bundleChanges[0].type).toBe('modify');
      expect(diff.bundleChanges[0].fieldChanges).toHaveLength(1);
      expect(diff.bundleChanges[0].fieldChanges![0].type).toBe('add');
      expect(diff.bundleChanges[0].fieldChanges![0].fieldName).toBe('name');
    });

    it('should detect field removals', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.bundleChanges[0].fieldChanges).toHaveLength(1);
      expect(diff.bundleChanges[0].fieldChanges![0].type).toBe('remove');
      expect(diff.bundleChanges[0].fieldChanges![0].fieldName).toBe('email');
    });

    it('should detect index additions', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: [{ fieldName: 'email', type: 'hash' }]
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.indexChanges).toHaveLength(1);
      expect(diff.indexChanges[0].type).toBe('create');
      expect(diff.indexChanges[0].bundleName).toBe('User');
      expect(diff.indexChanges[0].index.fieldName).toBe('email');
    });

    it('should detect index deletions', () => {
      const localSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      const serverSchema: BundleDefinition[] = [
        {
          name: 'User',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true }
          ],
          indexes: [{ fieldName: 'email', type: 'hash' }]
        }
      ];

      const diff = schemaManager.compareSchemas(localSchema, serverSchema);

      expect(diff.hasChanges).toBe(true);
      expect(diff.indexChanges).toHaveLength(1);
      expect(diff.indexChanges[0].type).toBe('delete');
    });
  });
});
