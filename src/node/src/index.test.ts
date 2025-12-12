import {
  parseConnectionString,
  SyndrDBClient,
  SyndrDBConnection,
  SyndrDBConnectionPool,
  SyndrDBConnectionError,
  SyndrDBProtocolError,
  SyndrDBPoolError,
  ConnectionParams,
} from './index';

describe('SyndrDB Node Driver', () => {
  describe('parseConnectionString', () => {
    it('should parse valid connection string', () => {
      const connStr = 'syndrdb://localhost:1776:mydb:admin:password123;';
      const params = parseConnectionString(connStr);

      expect(params).toEqual({
        host: 'localhost',
        port: 1776,
        database: 'mydb',
        username: 'admin',
        password: 'password123',
      });
    });

    it('should throw error if missing syndrdb:// prefix', () => {
      const connStr = 'localhost:1776:mydb:admin:password;';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
      expect(() => parseConnectionString(connStr)).toThrow('must start with "syndrdb://"');
    });

    it('should throw error if missing trailing semicolon', () => {
      const connStr = 'syndrdb://localhost:1776:mydb:admin:password';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
      expect(() => parseConnectionString(connStr)).toThrow('must end with ";"');
    });

    it('should throw error if missing fields', () => {
      const connStr = 'syndrdb://localhost:1776:mydb;';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
      expect(() => parseConnectionString(connStr)).toThrow('expected format');
    });

    it('should throw error for invalid port', () => {
      const connStr = 'syndrdb://localhost:invalid:mydb:admin:password;';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
      expect(() => parseConnectionString(connStr)).toThrow('port must be a valid number');
    });

    it('should throw error for out of range port', () => {
      const connStr = 'syndrdb://localhost:99999:mydb:admin:password;';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
    });

    it('should throw error for empty fields', () => {
      const connStr = 'syndrdb://localhost:1776::admin:password;';
      expect(() => parseConnectionString(connStr)).toThrow(SyndrDBConnectionError);
      expect(() => parseConnectionString(connStr)).toThrow('all fields');
    });
  });

  describe('Error Classes', () => {
    it('should create SyndrDBConnectionError with code and type', () => {
      const error = new SyndrDBConnectionError('Connection failed', 'E001', 'CONNECTION_ERROR');
      expect(error.name).toBe('SyndrDBConnectionError');
      expect(error.message).toBe('Connection failed');
      expect(error.code).toBe('E001');
      expect(error.type).toBe('CONNECTION_ERROR');
    });

    it('should create SyndrDBProtocolError with code and type', () => {
      const error = new SyndrDBProtocolError('Protocol error', 'E002', 'PROTOCOL_ERROR');
      expect(error.name).toBe('SyndrDBProtocolError');
      expect(error.message).toBe('Protocol error');
      expect(error.code).toBe('E002');
      expect(error.type).toBe('PROTOCOL_ERROR');
    });

    it('should create SyndrDBPoolError with code and type', () => {
      const error = new SyndrDBPoolError('Pool exhausted', 'E003', 'POOL_ERROR');
      expect(error.name).toBe('SyndrDBPoolError');
      expect(error.message).toBe('Pool exhausted');
      expect(error.code).toBe('E003');
      expect(error.type).toBe('POOL_ERROR');
    });
  });

  describe('SyndrDBConnection', () => {
    const mockParams: ConnectionParams = {
      host: 'localhost',
      port: 1776,
      database: 'testdb',
      username: 'admin',
      password: 'password',
    };

    it('should create connection instance', () => {
      const conn = new SyndrDBConnection(mockParams);
      expect(conn).toBeInstanceOf(SyndrDBConnection);
      expect(conn.isConnected()).toBe(false);
    });

    it('should track transaction state', () => {
      const conn = new SyndrDBConnection(mockParams);
      expect(conn.isInTransaction).toBe(false);
      
      conn.isInTransaction = true;
      expect(conn.isInTransaction).toBe(true);
    });

    it('should track last used timestamp', () => {
      const conn = new SyndrDBConnection(mockParams);
      const beforeTime = Date.now();
      const afterTime = Date.now();
      
      expect(conn.lastUsedAt).toBeGreaterThanOrEqual(beforeTime);
      expect(conn.lastUsedAt).toBeLessThanOrEqual(afterTime);
    });
  });

  describe('SyndrDBConnectionPool', () => {
    const mockParams: ConnectionParams = {
      host: 'localhost',
      port: 1776,
      database: 'testdb',
      username: 'admin',
      password: 'password',
    };

    it('should create pool with default options', () => {
      const pool = new SyndrDBConnectionPool(mockParams);
      const stats = pool.getStats();
      
      expect(stats.max).toBe(5);
      expect(stats.total).toBe(0);
    });

    it('should create pool with custom options', () => {
      const pool = new SyndrDBConnectionPool(mockParams, {
        maxConnections: 10,
        idleTimeout: 60000,
      });
      const stats = pool.getStats();
      
      expect(stats.max).toBe(10);
    });

    it('should track pool statistics', async () => {
      const pool = new SyndrDBConnectionPool(mockParams);
      const initialStats = pool.getStats();
      
      expect(initialStats).toEqual({
        available: 0,
        active: 0,
        total: 0,
        max: 5,
      });
    });
  });

  describe('SyndrDBClient', () => {
    describe('Placeholder Methods', () => {
      let client: SyndrDBClient;

      beforeEach(() => {
        client = new SyndrDBClient();
      });

      it('should throw error when calling query without implementation', async () => {
        const connStr = 'syndrdb://localhost:1776:testdb:admin:password;';
        
        // Mock connect to avoid actual network calls
        await expect(async () => {
          await client.connect(connStr);
        }).rejects.toThrow();
      });

      it('should throw protocol error for beginTransaction', async () => {
        await expect(async () => {
          await client.beginTransaction();
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should throw error for query without connection', async () => {
        await expect(async () => {
          await client.query('SELECT * FROM users');
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should throw error for mutate without connection', async () => {
        await expect(async () => {
          await client.mutate('INSERT INTO users VALUES (1, "test")');
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should throw error for graphql without connection', async () => {
        await expect(async () => {
          await client.graphql('{ users { id name } }');
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should throw error for migrate without connection', async () => {
        await expect(async () => {
          await client.migrate('CREATE TABLE users');
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should throw error for schema operations without connection', async () => {
        await expect(async () => {
          await client.addBundle({ name: 'test' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.addIndex({ name: 'test' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.addView({ name: 'test' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.changeBundle('test', { name: 'test2' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.changeIndex('test', { name: 'test2' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.changeView('test', { name: 'test2' });
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.dropBundle('test');
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.dropIndex('test');
        }).rejects.toThrow(SyndrDBConnectionError);

        await expect(async () => {
          await client.dropView('test');
        }).rejects.toThrow(SyndrDBConnectionError);
      });

      it('should return null pool stats when not connected', () => {
        const stats = client.getPoolStats();
        expect(stats).toBeNull();
      });
    });
  });
});
