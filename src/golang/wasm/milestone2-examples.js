// SyndrDB Milestone 2 - Parameterized Queries & Transactions Example
// This example demonstrates the new WASM APIs for secure parameterized queries and transactions

// Example 1: Parameterized Queries with Prepare/Execute
async function parameterizedQueryExample() {
  console.log('=== Parameterized Query Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  // Create and connect client
  await SyndrDB.createClient({
    defaultTimeoutMs: 5000,
    debugMode: true
  });
  
  await SyndrDB.connect('syndrdb://localhost:7437/mydb');
  
  try {
    // Prepare a statement with placeholders
    const stmt = await SyndrDB.prepare('get_user_by_name', 
      'SELECT * FROM "Users" WHERE "Name" = $1'
    );
    console.log('Prepared statement:', stmt);
    
    // Execute with different parameters
    const user1 = await SyndrDB.executeStatement(stmt.statementId, ['Alice']);
    console.log('User 1:', user1);
    
    const user2 = await SyndrDB.executeStatement(stmt.statementId, ['Bob']);
    console.log('User 2:', user2);
    
    // Deallocate when done
    await SyndrDB.deallocateStatement(stmt.statementId);
    console.log('Statement deallocated');
    
  } catch (error) {
    console.error('Error:', error);
  }
}

// Example 2: QueryWithParams - Automatic Statement Management
async function queryWithParamsExample() {
  console.log('=== QueryWithParams Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    // Auto-prepare, execute, and deallocate in one call
    const results = await SyndrDB.queryWithParams(
      'SELECT * FROM "Books" WHERE "Author" = $1 AND "Year" >= $2',
      ['Strohschein', 2020]
    );
    console.log('Books:', results);
    
    // SQL injection protection - these are treated as literal strings
    const attackAttempt = await SyndrDB.queryWithParams(
      'SELECT * FROM "Users" WHERE "Name" = $1',
      ["' OR '1'='1"]
    );
    console.log('Injection attempt neutralized:', attackAttempt);
    
  } catch (error) {
    console.error('Error:', error);
  }
}

// Example 3: Transactions with Manual Control
async function transactionExample() {
  console.log('=== Transaction Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    // Start transaction
    const tx = await SyndrDB.beginTransaction();
    console.log('Transaction started:', tx.transactionId);
    
    try {
      // Execute queries within transaction
      await SyndrDB.queryWithParams(
        'UPDATE "Accounts" SET "Balance" = "Balance" - $1 WHERE "ID" = $2',
        [100, 'account1']
      );
      
      await SyndrDB.queryWithParams(
        'UPDATE "Accounts" SET "Balance" = "Balance" + $1 WHERE "ID" = $2',
        [100, 'account2']
      );
      
      // Commit if all succeed
      await SyndrDB.commitTransaction(tx.transactionId);
      console.log('Transaction committed');
      
    } catch (error) {
      // Rollback on error
      await SyndrDB.rollbackTransaction(tx.transactionId);
      console.log('Transaction rolled back');
      throw error;
    }
    
  } catch (error) {
    console.error('Transaction error:', error);
  }
}

// Example 4: InTransaction - Automatic Commit/Rollback
async function inTransactionExample() {
  console.log('=== InTransaction Helper Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    const result = await SyndrDB.inTransaction(async (tx) => {
      console.log('Inside transaction:', tx.transactionId);
      
      // All queries here are part of the transaction
      await SyndrDB.queryWithParams(
        'INSERT INTO "Orders" ("CustomerID", "Total") VALUES ($1, $2)',
        ['customer123', 99.99]
      );
      
      await SyndrDB.queryWithParams(
        'UPDATE "Inventory" SET "Stock" = "Stock" - $1 WHERE "ProductID" = $2',
        [1, 'product456']
      );
      
      // Return value from transaction
      return { orderId: '12345', status: 'completed' };
    });
    
    console.log('Transaction auto-committed, result:', result);
    
  } catch (error) {
    console.error('Transaction auto-rolled back:', error);
  }
}

// Example 5: Complex Query with Multiple Parameters
async function complexQueryExample() {
  console.log('=== Complex Query Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    // Query with multiple conditions
    const results = await SyndrDB.queryWithParams(
      `SELECT * FROM "Products" 
       WHERE "Category" = $1 
       AND "Price" >= $2 
       AND "Price" <= $3 
       AND "InStock" = $4 
       ORDER BY "Price"`,
      ['Electronics', 100, 500, true]
    );
    console.log('Filtered products:', results);
    
    // Query with date parameter
    const recentOrders = await SyndrDB.queryWithParams(
      'SELECT * FROM "Orders" WHERE "CreatedAt" >= $1',
      [new Date('2024-01-01').toISOString()]
    );
    console.log('Recent orders:', recentOrders);
    
  } catch (error) {
    console.error('Error:', error);
  }
}

// Example 6: Statement Reuse for Performance
async function statementReuseExample() {
  console.log('=== Statement Reuse Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    // Prepare once, execute many times (better performance)
    const stmt = await SyndrDB.prepare('search_users',
      'SELECT * FROM "Users" WHERE "Department" = $1'
    );
    
    const departments = ['Engineering', 'Sales', 'Marketing', 'HR'];
    
    for (const dept of departments) {
      const users = await SyndrDB.executeStatement(stmt.statementId, [dept]);
      console.log(`${dept} users:`, users);
    }
    
    // Clean up
    await SyndrDB.deallocateStatement(stmt.statementId);
    
  } catch (error) {
    console.error('Error:', error);
  }
}

// Example 7: Error Handling
async function errorHandlingExample() {
  console.log('=== Error Handling Example ===');
  
  const SyndrDB = window.SyndrDB;
  
  try {
    // Wrong parameter count
    await SyndrDB.queryWithParams(
      'SELECT * FROM "Users" WHERE "Name" = $1 AND "Age" = $2',
      ['Alice'] // Missing second parameter
    );
  } catch (error) {
    console.error('Parameter count mismatch:', error);
  }
  
  try {
    // Invalid statement name
    await SyndrDB.prepare('invalid-name!', 'SELECT * FROM "Users"');
  } catch (error) {
    console.error('Invalid statement name:', error);
  }
  
  try {
    // Statement not found
    await SyndrDB.executeStatement('non_existent_stmt', []);
  } catch (error) {
    console.error('Statement not found:', error);
  }
}

// Run all examples
async function runAllExamples() {
  try {
    await parameterizedQueryExample();
    await queryWithParamsExample();
    await complexQueryExample();
    await statementReuseExample();
    
    // Transaction examples
    await transactionExample();
    await inTransactionExample();
    
    await errorHandlingExample();
    
  } finally {
    // Cleanup
    await window.SyndrDB.disconnect();
    console.log('Disconnected');
  }
}

// Auto-run on page load
if (typeof window !== 'undefined') {
  window.addEventListener('load', () => {
    console.log('SyndrDB Milestone 2 Examples Ready');
    console.log('Run: runAllExamples() to test all features');
  });
}
