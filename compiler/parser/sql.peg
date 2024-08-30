{
  const reservedMap = {
    'ARRAY': true,
    'ALTER': true,
    'ALL': true,
    'ADD': true,
    'AND': true,
    'AS': true,
    'ASC': true,

    'BETWEEN': true,
    'BY': true,

    'CALL': true,
    'CASE': true,
    'CREATE': true,
    'CROSS': true,
    'CONTAINS': true,
    'CURRENT_DATE': true,
    'CURRENT_TIME': true,
    'CURRENT_TIMESTAMP': true,
    'CURRENT_USER': true,

    'DELETE': true,
    'DESC': true,
    'DISTINCT': true,
    'DROP': true,

    'ELSE': true,
    'END': true,
    'EXISTS': true,
    'EXPLAIN': true,

    'FALSE': true,
    'FROM': true,
    'FULL': true,
    'FOR': true,

    'GROUP': true,

    'HAVING': true,

    'IN': true,
    'INNER': true,
    'INSERT': true,
    'INTERSECT': true,
    'INTO': true,
    'IS': true,

    'JOIN': true,
    'JSON': true,

    'KEY': false,

    'LEFT': true,
    'LIKE': true,
    'LIMIT': true,
    'LOW_PRIORITY': true, // for lock table

    'NOT': true,
    'NULL': true,

    'ON': true,
    'OR': true,
    'ORDER': true,
    'OUTER': true,

    'PARTITION': true,
    'PIVOT': true,

    'RECURSIVE': true,
    'RENAME': true,
    'READ': true, // for lock table
    'RIGHT': false,

    'SELECT': true,
    'SESSION_USER': true,
    'SET': true,
    'SHOW': true,
    'SYSTEM_USER': true,

    'TABLE': true,
    'THEN': true,
    'TRUE': true,
    'TRUNCATE': true,
    // 'TYPE': true,   // reserved (MySQL)

    'UNION': true,
    'UPDATE': true,
    'USING': true,

    'VALUES': true,

    'WINDOW': true,
    'WITH': true,
    'WHEN': true,
    'WHERE': true,
    'WRITE': true, // for lock table

    'GLOBAL': true,
    // 'SESSION': true,
    'LOCAL': true,
    'PERSIST': true,
    'PERSIST_ONLY': true,
    'UNNEST': true,
  };

  const DATA_TYPES = {
    'BOOL': true,
    'BYTE': true,
    'DATE': true,
    'DATETIME': true,
    'FLOAT64': true,
    'INT64': true,
    'NUMERIC': true,
    'STRING': true,
    'TIME': true,
    'TIMESTAMP': true,
    'ARRAY': true,
    'STRUCT': true,
  }

  function getLocationObject() {
    return options.includeLocations ? {loc: location()} : {}
  }

  function createUnaryExpr(op, e) {
    return {
      type: 'unary_expr',
      operator: op,
      expr: e
    };
  }

  function createBinaryExpr(op, left, right) {
    return {
      type: 'binary_expr',
      operator: op,
      left: left,
      right: right,
      ...getLocationObject(),
    };
  }

  function isBigInt(numberStr) {
    const previousMaxSafe = BigInt(Number.MAX_SAFE_INTEGER)
    const num = BigInt(numberStr)
    if (num < previousMaxSafe) return false
    return true
  }

  function createList(head, tail, po = 3) {
    const result = [head];
    for (let i = 0; i < tail.length; i++) {
      delete tail[i][po].tableList
      delete tail[i][po].columnList
      result.push(tail[i][po]);
    }
    return result;
  }

  function createBinaryExprChain(head, tail) {
    let result = head;
    for (let i = 0; i < tail.length; i++) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3]);
    }
    return result;
  }

  function queryTableAlias(tableName) {
    const alias = tableAlias[tableName]
    if (alias) return alias
    if (tableName) return tableName
    return null
  }

  function columnListTableAlias(columnList) {
    const newColumnsList = new Set()
    const symbolChar = '::'
    for(let column of columnList.keys()) {
      const columnInfo = column.split(symbolChar)
      if (!columnInfo) {
        newColumnsList.add(column)
        break
      }
      if (columnInfo && columnInfo[1]) columnInfo[1] = queryTableAlias(columnInfo[1])
      newColumnsList.add(columnInfo.join(symbolChar))
    }
    return Array.from(newColumnsList)
  }

  function refreshColumnList(columnList) {
    const columns = columnListTableAlias(columnList)
    columnList.clear()
    columns.forEach(col => columnList.add(col))
  }

  const cmpPrefixMap = {
    '+': true,
    '-': true,
    '*': true,
    '/': true,
    '>': true,
    '<': true,
    '!': true,
    '=': true,

    //between
    'B': true,
    'b': true,
    //for is or in
    'I': true,
    'i': true,
    //for like
    'L': true,
    'l': true,
    //for not
    'N': true,
    'n': true
  };

  // used for dependency analysis
  let varList = [];

  const tableList = new Set();
  const columnList = new Set();
  const tableAlias = {};
}

start
  = __ n:(multiple_stmt) {
    return n
  }

multiple_stmt
  = head:stmt tail:(__ SEMICOLON __ stmt)* {
      const headAst = head && head.ast || head
      const cur = tail && tail.length && tail[0].length >= 4 ? [headAst] : headAst;
      for (let i = 0; i < tail.length; i++) {
        if(!tail[i][3] || tail[i][3].length === 0) continue;
        cur.push(tail[i][3] && tail[i][3].ast || tail[i][3]);
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: cur
      }
    }

stmt
  = query_statement / crud_stmt

crud_stmt
  = union_stmt
  / update_stmt
  / replace_insert_stmt
  / insert_no_columns_stmt
  / insert_into_set
  / delete_stmt
  / cmd_stmt
  / proc_stmts

update_stmt
  = KW_UPDATE    __
    t:table_ref_list __
    KW_SET       __
    l:set_list   __
    f:from_clause? __
    w:where_clause? __
    or:order_by_clause? __
    lc:limit_clause? {
      if (t) t.forEach(tableInfo => {
        const { db, as, table, join } = tableInfo
        const action = join ? 'select' : 'update'
        tableList.add(`${action}::${db}::${table}`)
      });
      if(f) f.forEach(info => {
        info.table && tableList.add(`update::${info.db}::${info.table}`);
      });
      if(l) {
        l.forEach(col => columnList.add(`update::${col.table}::${col.column}`));
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: 'update',
          table: t,
          set: l,
          where: w,
          orderby: or,
          limit: lc,
        }
      };
    }

delete_stmt
  = KW_DELETE    __
    t: table_ref_list? __
    f:from_clause? __
    w:where_clause? __
    or:order_by_clause? __
    l:limit_clause? {
      if(t) t.forEach(tt => tableList.add(`delete::${tt.db}::${tt.table}`));
     if(f) f.forEach(tableInfo => {
        const { db, as, table, join } = tableInfo
        const action = join ? 'select' : 'delete'
        if (table) tableList.add(`${action}::${db}::${table}`)
        if (!join) columnList.add(`delete::${table}::(.*)`);
      });
      if (t === null && f.length === 1) {
        const tableInfo = f[0]
        t = [{
          db: tableInfo.db,
          table: tableInfo.table,
          as: tableInfo.as,
          addition: true
        }]
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: 'delete',
          table: t,
          from: f,
          where: w,
          orderby: or,
          limit: l,
        }
      };
    }

replace_insert_stmt
  = ri:replace_insert       __
    KW_INTO?                 __
    t:table_name  __
    p:insert_partition? __ LPAREN __ c:column_list  __ RPAREN __
    v:insert_value_clause __
    odp:on_duplicate_update_stmt? {
      if (t) {
        tableList.add(`insert::${t.db}::${t.table}`)
        t.as = null
      }
      if (c) {
        let table = t && t.table || null
        if(Array.isArray(v)) {
          v.forEach((row, idx) => {
            if(row.value.length != c.length) {
              throw new Error(`Error: column count doesn't match value count at row ${idx+1}`)
            }
          })
        }
        c.forEach(c => columnList.add(`insert::${table}::${c}`));
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: ri,
          table: [t],
          columns: c,
          values: v,
          partition: p,
          on_duplicate_update: odp,
        }
      };
    }

insert_no_columns_stmt
  = ri:replace_insert       __
    ig:KW_IGNORE?  __
    it:KW_INTO?   __
    t:table_name  __
    p:insert_partition? __
    v:insert_value_clause __
    odp: on_duplicate_update_stmt? {
      if (t) {
        tableList.add(`insert::${t.db}::${t.table}`)
        columnList.add(`insert::${t.table}::(.*)`);
        t.as = null
      }
      const prefix = [ig, it].filter(v => v).map(v => v[0] && v[0].toLowerCase()).join(' ')
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: ri,
          table: [t],
          columns: null,
          values: v,
          partition: p,
          prefix,
          on_duplicate_update: odp,
        }
      };
    }

insert_into_set
  = ri:replace_insert __
    KW_INTO? __
    t:table_name  __
    p:insert_partition? __
    KW_SET       __
    l:set_list   __
    odp:on_duplicate_update_stmt? {
      if (t) {
        tableList.add(`insert::${t.db}::${t.table}`)
        columnList.add(`insert::${t.table}::(.*)`);
        t.as = null
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: ri,
          table: [t],
          columns: null,
          partition: p,
          set: l,
          on_duplicate_update: odp,
        }
      };
    }

cmd_stmt
  = analyze_stmt
  / attach_stmt
  / drop_stmt
  / create_stmt
  / truncate_stmt
  / rename_stmt
  / call_stmt
  / use_stmt
  / alter_stmt
  / set_stmt
  / lock_stmt
  / unlock_stmt
  / show_stmt
  / desc_stmt

proc_stmts
  = proc_stmt*

proc_stmt
  = &{ varList = []; return true; } __ s:(assign_stmt / return_stmt) {
      return { stmt: s, vars: varList };
    }

assign_stmt_list
  = head:assign_stmt tail:(__ COMMA __ assign_stmt)* {
    return createList(head, tail);
  }

assign_stmt
  = va:(var_decl / without_prefix_var_decl) __ s: (KW_ASSIGN / KW_ASSIGIN_EQUAL) __ e:proc_expr {
    return {
      type: 'assign',
      left: va,
      symbol: s,
      right: e
    };
  }


return_stmt
  = KW_RETURN __ e:proc_expr {
      return { type: 'return', expr: e };
    }

proc_expr
  = select_stmt
  / proc_join
  / proc_additive_expr
  / proc_array

proc_additive_expr
  = head:proc_multiplicative_expr
    tail:(__ additive_operator  __ proc_multiplicative_expr)* {
      return createBinaryExprChain(head, tail);
    }

proc_multiplicative_expr
  = head:proc_primary
    tail:(__ multiplicative_operator  __ proc_primary)* {
      return createBinaryExprChain(head, tail);
    }

proc_join
  = lt:var_decl __ op:join_op  __ rt:var_decl __ expr:on_clause {
      return {
        type: 'join',
        ltable: lt,
        rtable: rt,
        op: op,
        on: expr
      };
    }

proc_primary
  = literal
  / var_decl
  / proc_func_call
  / param
  / LPAREN __ e:proc_additive_expr __ RPAREN {
      e.parentheses = true;
      return e;
    }

proc_func_call
  = name:proc_func_name __ LPAREN __ l:proc_primary_list? __ RPAREN {
      //compatible with original func_call
      return {
        type: 'function',
        name: name,
        args: {
          type: 'expr_list',
          value: l
        }
      };
    }
  / name:proc_func_name {
    return {
        type: 'function',
        name: name,
        args: null
      };
  }

proc_primary_list
  = head:proc_primary tail:(__ COMMA __ proc_primary)* {
      return createList(head, tail);
    }

proc_array
  = LBRAKE __ l:proc_primary_list __ RBRAKE {
    return { type: 'array', value: l, brackets: true };
  }

set_list
  = head:set_item tail:(__ COMMA __ set_item)* {
      return createList(head, tail);
    }

/**
 * here only use `additive_expr` to support 'col1 = col1+2'
 * if you want to use lower operator, please use '()' like below
 * 'col1 = (col2 > 3)'
 */
set_item
  = tbl:(ident __ DOT)? __ c:column_without_kw __ '=' __ v:additive_expr {
      return { column: c, value: v, table: tbl && tbl[0] };
  }
  / tbl:(ident __ DOT)? __ c:column_without_kw __ '=' __ KW_VALUES __ LPAREN __ v:column_ref __ RPAREN {
      return { column: c, value: v, table: tbl && tbl[0], keyword: 'values' };
  }

replace_insert
  = KW_INSERT   { return 'insert'; }
  / KW_REPLACE  { return 'replace'; }

insert_partition
  = KW_PARTITION __ LPAREN __ head:ident_name tail:(__ COMMA __ ident_name)* __ RPAREN {
      return createList(head, tail)
    }
  / KW_PARTITION __ v: value_item {
    return v
  }

insert_value_clause
  = value_clause
  / select_stmt_nake

on_duplicate_update_stmt
  = KW_ON __ 'DUPLICATE'i __ KW_KEY __ KW_UPDATE __ s:set_list {
    return {
      keyword: 'on duplicate key update',
      set: s
    }
  }

analyze_stmt
  = a:KW_ANALYZE __ t:table_name __ {
      tableList.add(`${a}::${t.db}::${t.table}`);
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a.toLowerCase(),
          table: t
        }
      };
    }

attach_stmt
  = a:KW_ATTACH __ db: KW_DATABASE __ e:expr __ as:KW_AS __ schema:ident __ {
      // tableList.add(`${a}::${t.db}::${t.table}`);
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a.toLowerCase(),
          database: db,
          expr: e,
          as: as && as[0].toLowerCase(),
          schema,
        }
      };
    }

drop_stmt
  = a:KW_DROP __
    r:KW_TABLE __
    t:table_ref_list {
      if(t) t.forEach(tt => tableList.add(`${a}::${tt.db}::${tt.table}`));
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a.toLowerCase(),
          keyword: r.toLowerCase(),
          name: t
        }
      };
    }
  / a:KW_DROP __
    r:KW_INDEX __
    i:column_ref __
    KW_ON __
    t:table_name __
    op:drop_index_opt? __ {
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a.toLowerCase(),
          keyword: r.toLowerCase(),
          name: i,
          table: t,
          options: op
        }
      };
    }

create_stmt
  = create_table_stmt
  / create_db_stmt
  / create_view_stmt

truncate_stmt
  = a:KW_TRUNCATE  __
    kw:KW_TABLE? __
    t:table_ref_list {
      if(t) t.forEach(tt => tableList.add(`${a}::${tt.db}::${tt.table}`));
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a.toLowerCase(),
          keyword: kw && kw.toLowerCase() || 'table',
          name: t
        }
      };
    }

rename_stmt
  = KW_RENAME  __
    KW_TABLE __
    t:table_to_list {
      t.forEach(tg => tg.forEach(dt => dt.table && tableList.add(`rename::${dt.db}::${dt.table}`)))
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: 'rename',
          table: t
        }
      };
    }

 call_stmt
  = KW_CALL __
  e: proc_func_call {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'call',
        expr: e
      }
    }
  }

use_stmt
  = KW_USE  __
    d:ident {
      tableList.add(`use::${d}::null`);
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: 'use',
          db: d
        }
      };
    }

alter_stmt
  = alter_table_stmt

set_stmt
  = KW_SET __
  kw: (KW_GLOBAL / KW_SESSION / KW_LOCAL / KW_PERSIST / KW_PERSIST_ONLY)? __
  a: assign_stmt_list {
    a.keyword = kw
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'set',
        keyword: kw,
        expr: a
      }
    }
  }

lock_stmt
  = KW_LOCK __ KW_TABLES __ ltl:lock_table_list {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'lock',
        keyword: 'tables',
        tables: ltl
      }
    }
  }

unlock_stmt
  = KW_UNLOCK __ KW_TABLES {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'unlock',
        keyword: 'tables'
      }
    }
  }

show_stmt
  = KW_SHOW __ t:('BINARY'i / 'MASTER'i) __ 'LOGS'i {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'show',
        suffix: 'logs',
        keyword: t.toLowerCase()
      }
    }
  }
  / KW_SHOW __ 'BINLOG'i __ 'EVENTS'i __ ins:in_op_right? __ from: from_clause? __ limit: limit_clause? {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'show',
        suffix: 'events',
        keyword: 'binlog',
        in: ins,
        from,
        limit,
      }
    }
  }
  / KW_SHOW __ k:(('CHARACTER'i __ 'SET'i) / 'COLLATION'i) __ e:(like_op_right / where_clause)? {
    let keyword = Array.isArray(k) && k || [k]
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'show',
        suffix: keyword[2] && keyword[2].toLowerCase(),
        keyword: keyword[0].toLowerCase(),
        expr: e
      }
    }
  }
  / show_grant_stmt

 desc_stmt
  = (KW_DESC / KW_DESCRIBE) __ t:ident {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'desc',
        table: t
      }
    };
  }

var_decl
  = p: KW_VAR_PRE d: without_prefix_var_decl {
    //push for analysis
    return {
      type: 'var',
      ...d,
      prefix: p
    };
  }

without_prefix_var_decl
  = name:ident_name m:mem_chain {
    //push for analysis
    varList.push(name);
    return {
      type: 'var',
      name: name,
      members: m,
      prefix: null,
    };
  }
  / n:literal_numeric {
    return {
      type: 'var',
      name: n.value,
      members: [],
      quoted: null,
      prefix: null,
    }
  }

value_item
  = LPAREN __ l:expr_list  __ RPAREN {
      return l;
    }

value_clause
  = KW_VALUES __ l:value_list  { return l; }

drop_index_opt
  = head:(ALTER_ALGORITHM / ALTER_LOCK) tail:(__ (ALTER_ALGORITHM / ALTER_LOCK))* {
    return createList(head, tail, 1)
  }

if_not_exists_stmt
  = 'IF'i __ KW_NOT __ KW_EXISTS {
    return 'IF NOT EXISTS'
  }

create_table_stmt
  = a:KW_CREATE __
    or:(KW_OR __ KW_REPLACE)? __
    tp:(KW_TEMP / KW_TEMPORARY)? __
    KW_TABLE __
    ife:if_not_exists_stmt? __
    t:table_name __
    c:create_table_definition? __
    to:table_options? __
    as:KW_AS? __
    qe:union_stmt? {
      if(t) tableList.add(`create::${t.db}::${t.table}`)
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a[0].toLowerCase(),
          keyword: 'table',
          temporary: tp && tp[0].toLowerCase(),
          if_not_exists:ife,
          table: [t],
          replace: or && 'or replace',
          as: as && as[0].toLowerCase(),
          query_expr: qe && qe.ast,
          create_definitions: c,
          table_options: to
        }
      }
    }
  / a:KW_CREATE __
    tp:KW_TEMPORARY? __
    KW_TABLE __
    ife:if_not_exists_stmt? __
    t:table_ref_list __
    lt:create_like_table {
      if(t) t.forEach(tt => tableList.add(`create::${tt.db}::${tt.table}`));
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a[0].toLowerCase(),
          keyword: 'table',
          temporary: tp && tp[0].toLowerCase(),
          if_not_exists:ife,
          table: t,
          like: lt
        }
      }
    }

create_db_stmt
  = a:KW_CREATE __
    k:(KW_DATABASE / KW_SCHEMA) __
    ife:if_not_exists_stmt? __
    t:proc_func_name __
    c:create_db_definition? {
      const keyword = k.toLowerCase()
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: a[0].toLowerCase(),
          keyword,
          if_not_exists:ife,
          [keyword]: { db: t.schema, schema: t.name },
          create_definitions: c,
        }
      }
    }

view_with
  = KW_WITH __ c:("CASCADED"i / "LOCAL"i) __ "CHECK"i __ "OPTION" {
    // => string
    return `with ${c.toLowerCase()} check option`
  }
  / KW_WITH __ "CHECK"i __ "OPTION" {
    // => string
    return 'with check option'
  }

with_view_option
  = 'check_option'i __ KW_ASSIGIN_EQUAL __ t:("CASCADED"i / "LOCAL"i) {
    return  { type: 'check_option', value: t, symbol: '=' }
  }
  / k:('security_barrier'i / 'security_invoker'i) __ KW_ASSIGIN_EQUAL __ t:literal_bool {
    return { type: k.toLowerCase(), value: t.value ? 'true' : 'false', symbol: '=' }
  }
with_view_options
  = head:with_view_option tail:(__ COMMA __ with_view_option)* {
      return createList(head, tail);
    }
create_view_stmt
  = a:KW_CREATE __ or:(KW_OR __ KW_REPLACE)? __ tp:(KW_TEMP / KW_TEMPORARY)? __ r:KW_RECURSIVE? __
  KW_VIEW __ v:table_name __ c:(LPAREN __ column_list __ RPAREN)? __ wo:(KW_WITH __ LPAREN __ with_view_options __ RPAREN)? __
  KW_AS __ s:select_stmt __ w:view_with? {
    v.view = v.table
    delete v.table
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: a[0].toLowerCase(),
        keyword: 'view',
        replace: or && 'or replace',
        temporary: tp && tp[0].toLowerCase(),
        recursive: r && r.toLowerCase(),
        columns: c && c[2],
        select: s,
        view: v,
        with_options: wo && wo[4],
        with: w,
      }
    }
  }

alter_table_stmt
  = KW_ALTER  __
    KW_TABLE __
    t:table_ref_list __
    e:alter_action_list {
      if (t && t.length > 0) t.forEach(table => tableList.add(`alter::${table.db}::${table.table}`));
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: {
          type: 'alter',
          table: t,
          expr: e
        }
      };
    }

lock_table_list
  = head:lock_table tail:(__ COMMA __ lock_table)* {
    return createList(head, tail);
  }

show_grant_stmt
  = KW_SHOW __ 'GRANTS'i __ f:show_grant_for? {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        type: 'show',
        keyword: 'grants',
        for: f,
      }
    }
  }

mem_chain
  = l:('.' ident_name)* {
    const s = [];
    for (let i = 0; i < l.length; i++) {
      s.push(l[i][1]);
    }
    return s;
  }

value_list
  = head:value_item tail:(__ COMMA __ value_item)* {
      return createList(head, tail);
    }

ALTER_ALGORITHM
  = "ALGORITHM"i __ s:KW_ASSIGIN_EQUAL? __ val:("DEFAULT"i / "INSTANT"i / "INPLACE"i / "COPY"i) {
    return {
      type: 'alter',
      keyword: 'algorithm',
      resource: 'algorithm',
      symbol: s,
      algorithm: val
    }
  }

ALTER_LOCK
  = "LOCK"i __ s:KW_ASSIGIN_EQUAL? __ val:("DEFAULT"i / "NONE"i / "SHARED"i / "EXCLUSIVE"i) {
    return {
      type: 'alter',
      keyword: 'lock',
      resource: 'lock',
      symbol: s,
      lock: val
    }
  }

create_table_definition
  = LPAREN __ head:create_definition tail:(__ COMMA __ create_definition)* __ RPAREN {
      return createList(head, tail);
    }

create_definition
  = create_column_definition
  / create_index_definition
  / create_fulltext_spatial_index_definition

column_definition_opt
  = n:(literal_not_null / literal_null) {
    if (n && !n.value) n.value = 'null'
    return { nullable: n }
  }
  / d:default_expr {
    return { default_val: d }
  }
  / a:('AUTO_INCREMENT'i) {
    return { auto_increment: a.toLowerCase() }
  }
  / 'UNIQUE'i __ k:('KEY'i)? {
    const sql = ['unique']
    if (k) sql.push(k)
    return { unique: sql.join(' ').toLowerCase('') }
  }
  / p:('PRIMARY'i)? __ 'KEY'i {
    const sql = []
    if (p) sql.push('primary')
    sql.push('key')
    return { primary_key: sql.join(' ').toLowerCase('') }
  }
  / co:keyword_comment {
    return { comment: co }
  }
  / ca:collate_expr {
    return { collate: ca }
  }
  / cf:column_format {
    return { column_format: cf }
  }
  / s:storage {
    return { storage: s }
  }
  / re:reference_definition {
    return { reference_definition: re }
  }

column_definition_opt_list
  = head:column_definition_opt __ tail:(__ column_definition_opt)* {
    let opt = head
    for (let i = 0; i < tail.length; i++) {
      opt = { ...opt, ...tail[i][1] }
    }
    return opt
  }

create_column_definition
  = c:column_ref __
    d:data_type __
    cdo:column_definition_opt_list? {
      columnList.add(`create::${c.table}::${c.column}`)
      return {
        column: c,
        definition: d,
        resource: 'column',
        ...(cdo || {})
      }
    }

table_options
  = head:table_option tail:(__ COMMA? __ table_option)* {
    return createList(head, tail)
  }

create_like_table
  = create_like_table_simple
  / LPAREN __ e:create_like_table  __ RPAREN {
      e.parentheses = true;
      return e;
  }

create_db_definition
  = head:create_option_character_set tail:(__ create_option_character_set)* {
    return createList(head, tail, 1)
  }

alter_action_list
  = head:alter_action tail:(__ COMMA __ alter_action)* {
      return createList(head, tail);
    }

lock_table
  = t:table_base __ lt:lock_type {
    tableList.add(`lock::${t.db}::${t.table}`)
    return {
      table: t,
      lock_type: lt
    }
  }

show_grant_for
  = 'FOR'i __ n:ident __ h:(KW_VAR__PRE_AT __ ident)? __ u:show_grant_for_using? {
    return {
      user: n,
      host: h && h[2],
      role_list: u
    }
  }

create_constraint_definition
  = create_constraint_primary
  / create_constraint_unique
  / create_constraint_foreign
  / create_constraint_check

create_index_definition
  = kc:(KW_INDEX / KW_KEY) __
    c:column? __
    t:index_type? __
    de:cte_column_definition __
    id:index_options? __
    {
      return {
        index: c,
        definition: de,
        keyword: kc.toLowerCase(),
        index_type: t,
        resource: 'index',
        index_options: id,
      }
    }

create_fulltext_spatial_index_definition
  = p: (KW_FULLTEXT / KW_SPATIAL) __
    kc:(KW_INDEX / KW_KEY)? __
    c:column? __
    de: cte_column_definition __
    id: index_options? __
     {
      return {
        index: c,
        definition: de,
        keyword: kc && `${p.toLowerCase()} ${kc.toLowerCase()}` || p.toLowerCase(),
        index_options: id,
        resource: 'index',
      }
    }

default_expr
  = KW_DEFAULT __ ce:expr {
    return {
      type: 'default',
      value: ce
    }
  }

keyword_comment
  = k:KW_COMMENT __ s:KW_ASSIGIN_EQUAL? __ c:literal_string {
    return {
      type: k.toLowerCase(),
      keyword: k.toLowerCase(),
      symbol: s,
      value: c,
    }
  }

collate_expr
  = KW_COLLATE __ ca:ident_name __ s:KW_ASSIGIN_EQUAL __ t:ident {
    return {
      type: 'collate',
      keyword: 'collate',
      collate: {
        name: ca,
        symbol: s,
        value: t
      }
    }
  }
  / KW_COLLATE __ s:KW_ASSIGIN_EQUAL? __ ca:ident {
    return {
      type: 'collate',
      keyword: 'collate',
      collate: {
        name: ca,
        symbol: s,
      }
    }
  }

column_format
  = k:'COLUMN_FORMAT'i __ f:('FIXED'i / 'DYNAMIC'i / 'DEFAULT'i) {
    return {
      type: 'column_format',
      value: f.toLowerCase()
    }
  }

storage
  = k:'STORAGE'i __ s:('DISK'i / 'MEMORY'i) {
    return {
      type: 'storage',
      value: s.toLowerCase()
    }
  }

reference_definition
  = kc:KW_REFERENCES __
  t:table_ref_list __
  de:cte_column_definition __
  m:('MATCH FULL'i / 'MATCH PARTIAL'i / 'MATCH SIMPLE'i)? __
  od:on_reference? __
  ou:on_reference? {
    const on_action = []
    return {
        definition: de,
        table: t,
        keyword: kc.toLowerCase(),
        match: m && m.toLowerCase(),
        on_action: [od, ou].filter(v => v)
      }
  }
  / oa:on_reference {
    return {
      on_action: [oa]
    }
  }

table_option_list_item
  = k:('expiration_timestamp'i / 'partition_expiration_days'i / 'require_partition_filter'i / 'kms_key_name'i / 'friendly_name'i / 'description'i / 'labels'i / 'default_rounding_mode'i) __ s:(KW_ASSIGIN_EQUAL)? __ v:expr {
    return {
      keyword: k,
      symbol: '=',
      value: v
    }
  }
table_option_list
  = head:table_option_list_item tail:(__ COMMA __ table_option_list_item)* {
    return createList(head, tail);
  }
table_option
  = kw:('AUTO_INCREMENT'i / 'AVG_ROW_LENGTH'i / 'KEY_BLOCK_SIZE'i / 'MAX_ROWS'i / 'MIN_ROWS'i / 'STATS_SAMPLE_PAGES'i) __ s:(KW_ASSIGIN_EQUAL)? __ v:literal_numeric {
    return {
      keyword: kw.toLowerCase(),
      symbol: s,
      value: v.value
    }
  }
  / create_option_character_set
  / kw:(KW_COMMENT / 'CONNECTION'i) __ s:(KW_ASSIGIN_EQUAL)? __ c:literal_string {
    return {
      keyword: kw.toLowerCase(),
      symbol: s,
      value: `'${c.value}'`
    }
  }
  / kw:'COMPRESSION'i __ s:(KW_ASSIGIN_EQUAL)? __ v:("'"('ZLIB'i / 'LZ4'i / 'NONE'i)"'") {
    return {
      keyword: kw.toLowerCase(),
      symbol: s,
      value: v.join('').toUpperCase()
    }
  }
  / kw:'ENGINE'i __ s:(KW_ASSIGIN_EQUAL)? __ c:ident_name {
    return {
      keyword: kw.toLowerCase(),
      symbol: s,
      value: c.toUpperCase()
    }
  }
  / KW_PARTITION __ KW_BY __ v:expr {
    return {
      keyword: 'partition by',
      value: v
    }
  }
  / 'CLUSTER'i __ 'BY'i __ c:column_list {
    return {
      keyword: 'cluster by',
      value: c
    }
  }
  / 'OPTIONS'i __ LPAREN __ v:table_option_list __ RPAREN {
    return {
      keyword: 'options',
      parentheses: true,
      value: v
    }
  }

create_like_table_simple
  = KW_LIKE __ t: table_ref_list {
    return {
      type: 'like',
      table: t
    }
  }

create_option_character_set
  = kw:KW_DEFAULT? __ t:(create_option_character_set_kw / 'CHARSET'i / 'COLLATE'i) __ s:(KW_ASSIGIN_EQUAL)? __ v:ident_without_kw_type {
    return {
      keyword: kw && `${kw[0].toLowerCase()} ${t.toLowerCase()}` || t.toLowerCase(),
      symbol: s,
      value: v
    }
  }

alter_action
  = ALTER_ADD_COLUMN
  / ALTER_DROP_COLUMN
  / ALTER_RENAME_TABLE

lock_type
  = "READ"i __ s:("LOCAL"i)? {
    return {
      type: 'read',
      suffix: s && 'local'
    }
  }
  / p:("LOW_PRIORITY"i)? __ "WRITE"i {
    return {
      type: 'write',
      prefix: p && 'low_priority'
    }
  }

show_grant_for_using
  = KW_USING __ l:show_grant_for_using_list {
    return l
  }

show_grant_for_using_list
  = head:ident tail:(__ COMMA __ ident)* {
    return createList(head, tail);
  }


create_constraint_primary
  = kc:constraint_name? __
  p:('PRIMARY'i __ 'KEY'i) __
  t:index_type? __
  de:cte_column_definition __
  id:index_options? {
    return {
        constraint: kc && kc.constraint,
        definition: de,
        constraint_type: `${p[0].toLowerCase()} ${p[2].toLowerCase()}`,
        keyword: kc && kc.keyword,
        index_type: t,
        resource: 'constraint',
        index_options: id,
      }
  }

create_constraint_unique
  = kc:constraint_name? __
  u:KW_UNIQUE __
  p:(KW_INDEX / KW_KEY)? __
  i:column? __
  t:index_type? __
  de:cte_column_definition __
  id:index_options? {
    return {
        constraint: kc && kc.constraint,
        definition: de,
        constraint_type: p && `${u.toLowerCase()} ${p.toLowerCase()}` || u.toLowerCase(),
        keyword: kc && kc.keyword,
        index_type: t,
        index: i,
        resource: 'constraint',
        index_options: id
      }
  }

create_constraint_foreign
  = kc:constraint_name? __
  p:('FOREIGN KEY'i) __
  i:column? __
  de:cte_column_definition __
  id:reference_definition? {
    return {
        constraint: kc && kc.constraint,
        definition: de,
        constraint_type: p,
        keyword: kc && kc.keyword,
        index: i,
        resource: 'constraint',
        reference_definition: id
      }
  }

create_constraint_check
  = kc:constraint_name? __ u:'CHECK'i __ nfr:('NOT'i __ 'FOR'i __ 'REPLICATION'i __)? LPAREN __ c:expr __ RPAREN {
    return {
        constraint_type: u.toLowerCase(),
        keyword: kc && kc.keyword,
        constraint: kc && kc.constraint,
        index_type: nfr && { keyword: 'not for replication' },
        definition: [c],
        resource: 'constraint',
      }
  }

index_type
  = KW_USING __
  t:("BTREE"i / "HASH"i) {
    return {
      keyword: 'using',
      type: t.toLowerCase(),
    }
  }

cte_column_definition
  = LPAREN __ head:column tail:(__ COMMA __ column)* __ RPAREN {
      return createList(head, tail);
    }

index_options
  = head:index_option tail:(__ index_option)* {
    const result = [head];
    for (let i = 0; i < tail.length; i++) {
      result.push(tail[i][1]);
    }
    return result;
  }

index_option
  = k:KW_KEY_BLOCK_SIZE __ e:(KW_ASSIGIN_EQUAL)? __ kbs:literal_numeric {
    return {
      type: k.toLowerCase(),
      symbol: e,
      expr: kbs
    };
  }
  / index_type
  / "WITH"i __ "PARSER"i __ pn:ident_name {
    return {
      type: 'with parser',
      expr: pn
    }
  }
  / k:("VISIBLE"i / "INVISIBLE"i) {
    return {
      type: k.toLowerCase(),
      expr: k.toLowerCase()
    }
  }
  / keyword_comment

on_reference
  = KW_ON __ kw:(KW_DELETE / KW_UPDATE) __ ro:reference_option {
    // => { type: 'on delete' | 'on update'; value: reference_option; }
    return {
      type: `on ${kw[0].toLowerCase()}`,
      value: ro
    }
  }

create_option_character_set_kw
  = 'CHARACTER'i __ 'SET'i {
    return 'CHARACTER SET'
  }

ALTER_ADD_COLUMN
  = KW_ADD __
    kc:KW_COLUMN? __
    cd:create_column_definition {
      return {
        action: 'add',
        ...cd,
        keyword: kc,
        resource: 'column',
        type: 'alter',
      }
    }

ALTER_DROP_COLUMN
  = KW_DROP __
    kc:KW_COLUMN? __
    c:column_ref {
      return {
        action: 'drop',
        column: c,
        keyword: kc,
        resource: 'column',
        type: 'alter',
      }
    }

ALTER_RENAME_TABLE
  = KW_RENAME __
  kw:(KW_TO / KW_AS)? __
  tn:ident {
    return {
      action: 'rename',
      type: 'alter',
      resource: 'table',
      keyword: kw && kw[0].toLowerCase(),
      table: tn
    }
  }

constraint_name
  = kc:KW_CONSTRAINT __
  c:ident? {
    return {
      keyword: kc.toLowerCase(),
      constraint: c
    }
  }

reference_option
  = kw:KW_CURRENT_TIMESTAMP __ LPAREN __ l:expr_list? __ RPAREN {
    return {
      type: 'function',
      name: { name: [{ type: 'origin', value: kw }]},
      args: l
    }
  }
  / kc:('RESTRICT'i / 'CASCADE'i / 'SET NULL'i / 'NO ACTION'i / 'SET DEFAULT'i / KW_CURRENT_TIMESTAMP) {
    return {
      type: 'origin',
      value: kc.toLowerCase()
    }
  }

KW_UPDATE   = "UPDATE"i     !ident_start
KW_CREATE   = "CREATE"i     !ident_start
KW_DELETE   = "DELETE"i     !ident_start
KW_INSERT   = "INSERT"i     !ident_start
KW_ASSIGN = ':='
KW_ASSIGIN_EQUAL = '='
KW_RETURN   = 'return'i
KW_REPLACE  = "REPLACE"i    !ident_start
KW_ANALYZE  = "ANALYZE"i    !ident_start
KW_ATTACH   = "ATTACH"i     !ident_start
KW_DATABASE = "DATABASE"i   !ident_start
KW_RENAME   = "RENAME"i     !ident_start
KW_SHOW     = "SHOW"i       !ident_start
KW_DESCRIBE = "DESCRIBE"i   !ident_start
KW_VAR__PRE_AT = '@'
KW_VAR__PRE_AT_AT = '@@'
KW_VAR_PRE_DOLLAR = '$'
KW_VAR_PRE = KW_VAR__PRE_AT_AT / KW_VAR__PRE_AT / KW_VAR_PRE_DOLLAR
KW_TEMPORARY = "TEMPORARY"i !ident_start
KW_TEMP = "TEMP"i !ident_start
KW_SCHEMA   = "SCHEMA"i     !ident_start
KW_ALTER    = "ALTER"i      !ident_start
KW_SPATIAL  = "SPATIAL"i    !ident_start
KW_KEY_BLOCK_SIZE = "KEY_BLOCK_SIZE"i !ident_start

query_statement
  = query_expr
  / s:('(' __ select_stmt __ ')') {
      return {
        ...s[2],
        parentheses_symbol: true,
      }
    }

query_expr
  = s:union_stmt __ o:order_by_clause?  __ l:limit_clause? __
  {
    return {
      tableList: Array.from(tableList),
      columnList: columnListTableAlias(columnList),
      ast: {
        ...s.ast,
        _orderby: o,
        _limit: l,
        _parentheses: s._parentheses
      }
    }
  }

set_op
  = u:KW_UNION __ s:(KW_ALL / KW_DISTINCT)? {
    return s ? `union ${s.toLowerCase()}` : 'union'
  }
  / u:('INTERSECT'i / 'EXCEPT'i) __ s:KW_DISTINCT {
    return `${u.toLowerCase()} ${s.toLowerCase()}`
  }

union_stmt
  = union_stmt_nake
  / s:('(' __ union_stmt_nake __ ')') {
      return {
        ...s[2],
        _parentheses: true
      }
    }

union_stmt_nake
  = head:select_stmt tail:(__ set_op? __ select_stmt)* __ ob: order_by_clause? __ l:limit_clause?  {
      let cur = head
      for (let i = 0; i < tail.length; i++) {
        cur._next = tail[i][3]
        cur.set_op = tail[i][1]
        cur = cur._next
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: head
      }
    }
select_stmt
  = select_stmt_nake
  / s:('(' __ select_stmt __ ')') {
      return {
        ...s[2],
        parentheses_symbol: true
      }
    }

with_clause
  = KW_WITH __ head:cte_definition tail:(__ COMMA __ cte_definition)* {
      return createList(head, tail);
    }

cte_definition
  = name:(literal_string / ident_name) __ KW_AS __ LPAREN __ stmt:union_stmt __ RPAREN {
    if (typeof name === 'string') name = { type: 'default', value: name }
    return { name, stmt };
  }

select_stmt_nake
  = __ cte:with_clause? __ KW_SELECT ___
    sv:struct_value? __
    d:(KW_ALL / KW_DISTINCT)? __
    c:column_clause     __
    f:from_clause?      __
    fs:for_sys_time_as_of? __
    w:where_clause?     __
    g:group_by_clause?  __
    h:having_clause?    __
    q:qualify_clause? __
    o:order_by_clause?  __
    l:limit_clause? __
    win:window_clause? {
      if(Array.isArray(f)) f.forEach(info => info.table && tableList.add(`select::${info.db}::${info.table}`));
      return {
          type: 'select',
          as_struct_val: sv,
          distinct: d,
          columns: c,
          from: f,
          for_sys_time_as_of: fs,
          where: w,
          with: cte,
          groupby: g,
          having: h,
          qualify: q,
          orderby: o,
          limit: l,
          window:win,
          ...getLocationObject()
      };
  }

for_sys_time_as_of
  = 'FOR'i __ 'SYSTEM_TIME'i __ 'AS'i __ 'OF'i __ e:expr {
    return {
      keyword: 'for system_time as of',
      expr: e
    }
  }
struct_value
  = a:KW_AS __ k:(KW_STRUCT / KW_VALUE) {
    return `${a[0].toLowerCase()} ${k.toLowerCase()}`
  }

expr_alias
  = e:binary_column_expr __ alias:alias_clause? {
      return { expr: e, as: alias, ...getLocationObject() };
    }

column_clause
  = c:columns_list __ COMMA? {
    return c
  }

columns_list
  = head:column_list_item tail:(__ COMMA __ column_list_item)* {
      return createList(head, tail);
    }

column_offset_expr_list
  = l:(LBRAKE __ (literal_numeric / literal_string) __ RBRAKE)+ {
    return l.map(item => ({ value: item[2] }))
  }
  / l:(LBRAKE __ (KW_OFFSET / KW_ORDINAL / KW_SAFE_OFFSET / KW_SAFE_ORDINAL) __ LPAREN __ (literal_numeric / literal_string) __ RPAREN __ RBRAKE)+ {
    return l.map(item => ({ name: item[2], value: item[6] }))
  }
column_offset_expr
  = n:expr __ l:column_offset_expr_list {
    return {
      expr: n,
      offset: l
    }
  }

column_list_item
  = p:(column_without_kw __ DOT)? STAR __ k:('EXCEPT'i / 'REPLACE'i) __ LPAREN __ c:columns_list __ RPAREN {
    const tbl = p && p[0]
    columnList.add(`select::${tbl}::(.*)`)
    return {
      expr_list: c,
      parentheses: true,
      expr: {
        type: 'column_ref',
        table: tbl,
        column: '*'
      },
      type: k.toLowerCase(),
      ...getLocationObject(),
    }
  }
  / head: (KW_ALL / (STAR !ident_start) / STAR) {
      columnList.add('select::null::(.*)')
      const item = {
        expr: {
          type: 'column_ref',
          table: null,
          column: '*'
        },
        as: null,
        ...getLocationObject()
      }
      return item
  }
  / tbl:column_without_kw __ DOT pro:((column_offset_expr / column_without_kw) __ DOT)? __ STAR {
      columnList.add(`select::${tbl}::(.*)`)
      let column = '*'
      const mid = pro && pro[0]
      if (typeof mid === 'string') column = `${mid}.*`
      if (mid && mid.expr && mid.offset) column = { ...mid, suffix: '.*' }
      return {
        expr: {
          type: 'column_ref',
          table: tbl,
          column,
        },
        as: null,
        ...getLocationObject()
      }
    }
  / c:column_offset_expr __ s:(DOT __ column_without_kw)? __ as:alias_clause? {
    if (s) c.suffix = `.${s[2]}`
    return {
        expr: {
          type: 'column_ref',
          table: null,
          column: c
        },
        as: as,
        ...getLocationObject()
      }
  }
  / expr_alias

alias_clause
  = KW_AS __ i:alias_ident { return i; }
  / KW_AS? __ i:column { return i; }

from_unnest_item
  = 'UNNEST'i __ LPAREN __ a:expr? __ RPAREN __ alias:alias_clause? __ wf:with_offset? {
    return {
      type: 'unnest',
      expr: a,
      parentheses: true,
      as:alias,
      with_offset: wf,
    }
  }

from_clause
  = KW_FROM __ l:table_ref_list __ op:pivot_operator? {
    if (l[0]) l[0].operator = op
    return l
  }

pivot_operator
  = KW_PIVOT __ LPAREN __ a:aggr_func_list __ 'FOR'i __ c:column_ref __ i:in_op_right __ RPAREN __ as:alias_clause? {
    i.operator = '='
    return {
      'type': 'pivot',
      'expr': a,
      column: c,
      in_expr: i,
      as,
    }
  }

with_offset
  = KW_WITH __ KW_OFFSET __ alias:alias_clause? {
    return {
      keyword: 'with offset as',
      as: alias
    }
  }
table_to_list
  = head:table_to_item tail:(__ COMMA __ table_to_item)* {
      return createList(head, tail);
    }

table_to_item
  = head:table_name __ KW_TO __ tail: (table_name) {
      return [head, tail]
    }

table_ref_list
  = head:table_base
    tail:table_ref* {
      tail.unshift(head);
      tail.forEach(tableInfo => {
        const { table, as } = tableInfo
        tableAlias[table] = table
        if (as) tableAlias[as] = table
        refreshColumnList(columnList)
      })
      return tail;
    }

table_ref
  = __ COMMA __ t:table_base { return t; }
  / __ t:table_join { return t; }


table_join
  = op:join_op __ t:table_base __ KW_USING __ LPAREN __ head:ident_name tail:(__ COMMA __ ident_name)* __ RPAREN {
      t.join = op;
      t.using = createList(head, tail);
      return t;
    }
  / op:join_op __ t:table_base __ expr:on_clause? {
      t.join = op;
      t.on   = expr;
      return t;
    }
  / op:(join_op / set_op) __ LPAREN __ stmt:union_stmt __ RPAREN __ alias:alias_clause? __ expr:on_clause? {
    stmt.parentheses = true;
    return {
      expr: stmt,
      as: alias,
      join: op,
      on: expr
    };
  }

hint
  = ([\@])([\{]) __ ident_name __ ([\=]) __ ident_name __ ([}])

tablesample
  = 'TABLESAMPLE'i __ ( 'BERNOULLI'i / 'RESERVOIR'i ) __ '(' __ number  __ ( 'PERCENT'i / 'ROWS'i ) __ ')'

//NOTE that, the table assigned to `var` shouldn't write in `table_join`
table_base
  = from_unnest_item
  / t:table_name
    ht:hint? __
	  ts:tablesample? __
	  alias:alias_clause? {
      if (t.type === 'var') {
        t.as = alias;
        return t;
      }
      return {
        ...t,
        as: alias,
        ...getLocationObject(),
      };
    }
  / LPAREN __ stmt:union_stmt __ RPAREN __ ts:tablesample? __ alias:alias_clause? {
      stmt.parentheses = true;
      return {
        expr: stmt,
        as: alias,
        ...getLocationObject(),
      };
    }

join_op
  = KW_LEFT __ KW_OUTER? __ KW_JOIN { return 'LEFT JOIN'; }
  / KW_RIGHT __ KW_OUTER? __ KW_JOIN { return 'RIGHT JOIN'; }
  / KW_FULL __ KW_OUTER? __ KW_JOIN { return 'FULL JOIN'; }
  / k:KW_CROSS __ KW_JOIN { return `${k[0].toUpperCase()} JOIN`; }
  / k:KW_INNER? __ KW_JOIN { return k ? `${k[0].toUpperCase()} JOIN` : 'JOIN'; }

table_name
  = db:ident_without_kw schema:(__ DOT __ ident_without_kw) tail:(__ DOT __ ident_without_kw) {
      const obj = { db: null, table: db };
      if (tail !== null) {
        obj.db = db;
        obj.catalog = db;
        obj.schema = schema[3];
        obj.table = tail[3];
      }
      return obj;
    }
  / dt:ident_without_kw tail:(__ DOT __ ident_without_kw)? {
      const obj = { db: null, table: dt };
      if (tail !== null) {
        obj.db = dt;
        obj.table = tail[3];
      }
      return obj;
    }
or_and_expr
	= head:expr tail:(__ (KW_AND / KW_OR) __ expr)* {
    const len = tail.length
    let result = head
    for (let i = 0; i < len; ++i) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3])
    }
    return result
  }

on_clause
  = KW_ON __ e:or_and_where_expr { return e; }

where_clause
  = KW_WHERE __ e:or_and_where_expr { return e; }

group_by_clause
  = KW_GROUP __ KW_BY __ e:expr_list {
    return {
      columns: e.value
    }
  }

having_clause
  = KW_HAVING __ e:or_and_where_expr { return e; }

qualify_clause
  = KW_QUALIFY __ e:expr { return e }

window_clause
  = KW_WINDOW __ l:named_window_expr_list {
    return {
      keyword: 'window',
      type: 'window',
      expr: l,
    }
  }

named_window_expr_list
  = head:named_window_expr tail:(__ COMMA __ named_window_expr)* {
      return createList(head, tail);
    }

named_window_expr
  = nw:ident_name __ KW_AS __ anw:as_window_specification {
    return {
      name: nw,
      as_window_specification: anw,
    }
  }

as_window_specification
  = n:ident_name { return n }
  / LPAREN __ ws:window_specification? __ RPAREN {
    return {
      window_specification: ws,
      parentheses: true
    }
  }

window_specification
  = n:ident? __
  bc:partition_by_clause? __
  l:order_by_clause? __
  w:window_frame_clause? {
    return {
      name: n,
      partitionby: bc,
      orderby: l,
      window_frame_clause: w
    }
  }

window_frame_clause
  = 'RANGE'i __ KW_BETWEEN 'UNBOUNDED'i __ 'PRECEDING'i __ KW_AND __ 'CURRENT'i __ 'ROW' {
    return 'range between unbounded preceding and current row'
  }
  / kw:KW_ROWS __ s:(window_frame_following / window_frame_preceding) {
    // => string
    return `rows ${s.value}`
  }
  / KW_ROWS __ KW_BETWEEN __ p:window_frame_preceding __ KW_AND __ f:window_frame_following {
    // => string
    return `rows between ${p.value} and ${f.value}`
  }

window_frame_following
  = s:window_frame_value __ c:('FOLLOWING'i / 'PRECEDING'i) {
    // => string
    s.value += ` ${c.toUpperCase()}`
    return s
  }
  / window_frame_current_row

window_frame_preceding
  = s:window_frame_value __ 'PRECEDING'i  {
    // => string
    s.value += ' PRECEDING'
    return s
  }
  / window_frame_current_row

window_frame_current_row
  = 'CURRENT'i __ 'ROW'i {
    // => { type: 'single_quote_string'; value: string }
    return { type: 'single_quote_string', value: 'current row', ...getLocationObject() }
  }

window_frame_value
  = s:'UNBOUNDED'i {
    // => literal_string
    return { type: 'single_quote_string', value: s.toUpperCase(), ...getLocationObject() }
  }
  / literal_numeric

partition_by_clause
  = KW_PARTITION __ KW_BY __ bc:column_clause { return bc; }

order_by_clause
  = KW_ORDER __ KW_BY __ l:order_by_list { return l; }

order_by_list
  = head:order_by_element tail:(__ COMMA __ order_by_element)* {
      return createList(head, tail);
    }

order_by_element
  = e:expr __
    c:('COLLATE'i __ literal_string)? __
    d:(KW_DESC / KW_ASC)? {
    const obj = { expr: e, type: d };
    return obj;
  }

number_or_param
  = literal_numeric
  / param

limit_clause
  = KW_LIMIT __ i1:(number_or_param) __ tail:((COMMA / KW_OFFSET) __ number_or_param)? {
      const res = [i1];
      if (tail) res.push(tail[2]);
      return {
        seperator: tail && tail[0] && tail[0].toLowerCase() || '',
        value: res,
        ...getLocationObject(),
      };
    }

/**
 * here only use `additive_expr` to support 'col1 = col1+2'
 * if you want to use lower operator, please use '()' like below
 * 'col1 = (col2 > 3)'
 */

expr_list
  = head:expr tail:(__ COMMA __ expr)* {
      const el = { type: 'expr_list' };
      el.value = createList(head, tail);
      return el;
    }

_expr
  = struct_expr
  / json_expr
  / or_expr
  / unary_expr
  / array_expr

expr
  = _expr / union_stmt

parentheses_list_expr
  = head:parentheses_expr tail:(__ COMMA __ parentheses_expr)* {
      return createList(head, tail);
    }

parentheses_expr
  = LPAREN __ c:column_clause __ RPAREN {
    return c
  }

array_expr
  = LBRAKE __ c:column_clause? __ RBRAKE {
    return {
      array_path: c,
      type: 'array',
      brackets: true,
      keyword: '',
    }
  }
  / s:(array_type / KW_ARRAY)? LBRAKE __ c:literal_list __ RBRAKE {
    return {
      definition: s,
      array_path: c.map(l => ({ expr: l, as: null })),
      type: 'array',
      keyword: s && 'array',
      brackets: true,
    }
  }
   / s:(array_type / KW_ARRAY)? __ l:(LBRAKE) __ c:(parentheses_list_expr / expr) __ r:(RBRAKE) {
    return {
      definition: s,
      expr_list: c,
      type: 'array',
      keyword: s && 'array',
      brackets: true,
      parentheses: false
    }
  }
  / s:(array_type / KW_ARRAY) __ l:(LPAREN) __ c:(parentheses_list_expr / expr) __ r:(RPAREN) {
    return {
      definition: s,
      expr_list: c,
      type: 'array',
      keyword: s && 'array',
      brackets: false,
      parentheses: true
    }
  }

json_expr
  = KW_JSON __ l:literal_list {
    return {
      type: 'json',
      keyword: 'json',
      expr_list: l
    }
  }

struct_expr
  = s:(struct_type / KW_STRUCT) __ LPAREN __ c:column_clause __ RPAREN {
    return {
      definition: s,
      expr_list: c,
      type: 'struct',
      keyword: s && 'struct',
      parentheses: true
    }
  }

unary_expr
  = op: additive_operator tail: (__ primary)+ {
    return createUnaryExpr(op, tail[0][1]);
  }

binary_column_expr
  = head:expr tail:(__ (KW_AND / KW_OR / LOGIC_OPERATOR) __ expr)* {
    const ast = head.ast
    if (ast && ast.type === 'select') {
      if (!(head.parentheses_symbol || head.parentheses || head.ast.parentheses || head.ast.parentheses_symbol) || ast.columns.length !== 1 || ast.columns[0].expr.column === '*') throw new Error('invalid column clause with select statement')
    }
    if (!tail || tail.length === 0) return head
    const len = tail.length
    let result = tail[len - 1][3]
    for (let i = len - 1; i >= 0; i--) {
      const left = i === 0 ? head : tail[i - 1][3]
      result = createBinaryExpr(tail[i][1], left, result)
    }
    return result
  }
or_and_where_expr
	= head:expr tail:(__ (KW_AND / KW_OR / COMMA) __ expr)* {
    const len = tail.length
    let result = head;
    let seperator = ''
    for (let i = 0; i < len; ++i) {
      if (tail[i][1] === ',') {
        seperator = ','
        if (!Array.isArray(result)) result = [result]
        result.push(tail[i][3])
      } else {
        result = createBinaryExpr(tail[i][1], result, tail[i][3]);
      }
    }
    if (seperator === ',') {
      const el = { type: 'expr_list' }
      el.value = result
      return el
    }
    return result
  }

or_expr
  = head:and_expr tail:(___ KW_OR __ and_expr)* {
      return createBinaryExprChain(head, tail);
    }

and_expr
  = head:not_expr tail:(___ KW_AND __ not_expr)* {
      return createBinaryExprChain(head, tail);
    }

//here we should use `NOT` instead of `comparision_expr` to support chain-expr
not_expr
  = comparison_expr
  / exists_expr
  / (KW_NOT / "!" !"=") __ expr:not_expr {
      return createUnaryExpr('NOT', expr);
    }

comparison_expr
  = left:additive_expr __ rh:comparison_op_right? {
      if (rh === null) return left;
      else if (rh.type === 'arithmetic') return createBinaryExprChain(left, rh.tail);
      else return createBinaryExpr(rh.op, left, rh.right);
    }
  / literal_string
  / column_ref

exists_expr
  = op:exists_op __ LPAREN __ stmt:union_stmt __ RPAREN {
    stmt.parentheses = true;
    return createUnaryExpr(op, stmt);
  }

exists_op
  = nk:(KW_NOT __ KW_EXISTS) { return nk[0] + ' ' + nk[2]; }
  / KW_EXISTS

comparison_op_right
  = arithmetic_op_right
  / in_op_right
  / between_op_right
  / is_op_right
  / like_op_right

arithmetic_op_right
  = l:(__ arithmetic_comparison_operator __ (additive_expr))+ {
      return { type: 'arithmetic', tail: l };
    }

arithmetic_comparison_operator
  = ">=" / ">" / "<=" / "<>" / "<" / "=" / "!="

is_op_right
  = KW_IS __ right:additive_expr {
      return { op: 'IS', right: right };
    }
  / (KW_IS __ KW_NOT) __ right:additive_expr {
      return { op: 'IS NOT', right: right };
  }

between_op_right
  = op:between_or_not_between_op __  begin:additive_expr __ KW_AND __ end:additive_expr {
      return {
        op: op,
        right: {
          type: 'expr_list',
          value: [begin, end]
        }
      };
    }

between_or_not_between_op
  = nk:(KW_NOT __ KW_BETWEEN) { return nk[0] + ' ' + nk[2]; }
  / KW_BETWEEN

like_op
  = nk:(KW_NOT __ KW_LIKE) { return nk[0] + ' ' + nk[2]; }
  / KW_LIKE

in_op
  = nk:(KW_NOT __ KW_IN) { return nk[0] + ' ' + nk[2]; }
  / KW_IN

like_op_right
  = op:like_op __ right:(literal / comparison_expr) {
      return { op: op, right: right };
    }

in_op_right
  = op:in_op __ LPAREN  __ l:expr_list __ RPAREN {
      return { op: op, right: l };
    }
  / op:in_op __ e:(literal_string / from_unnest_item) {
      return { op: op, right: e };
    }

additive_expr
  = head:multiplicative_expr
    tail:(__ additive_operator  __ multiplicative_expr)* {
      if (tail && tail.length && head.type === 'column_ref' && head.column === '*') throw new Error(JSON.stringify({
        message: 'args could not be star column in additive expr',
        ...getLocationObject(),
      }))
      return createBinaryExprChain(head, tail);
    }

additive_operator
  = "+" / "-"

multiplicative_expr
  = head:unary_expr_or_primary
    tail:(__ (multiplicative_operator / LOGIC_OPERATOR)  __ unary_expr_or_primary)* {
      return createBinaryExprChain(head, tail)
    }

multiplicative_operator
  = "*" / "/" / "%"

primary
  = array_expr
  / aggr_func
  / func_call
  / struct_expr
  / json_expr
  / cast_expr
  / literal
  / case_expr
  / interval_expr
  / column_ref
  / param
  / LPAREN __ list:or_and_where_expr __ RPAREN {
        list.parentheses = true;
        return list;
    }

unary_expr_or_primary
  = primary
  / op:(unary_operator) tail:(__ unary_expr_or_primary) {
    // if (op === '!') op = 'NOT'
    return createUnaryExpr(op, tail[1])
  }

unary_operator
  = '!' / '-' / '+' / '~'

interval_expr
  = KW_INTERVAL __
    e:expr __
    u: interval_unit {
      return {
        type: 'interval',
        expr: e,
        unit: u.toLowerCase(),
      }
    }

case_expr
  = KW_CASE __
    condition_list:case_when_then_list __
    otherwise:case_else? __
    KW_END __ KW_CASE? {
      if (otherwise) condition_list.push(otherwise);
      return {
        type: 'case',
        expr: null,
        args: condition_list
      };
    }
  / KW_CASE __
    expr:expr __
    condition_list:case_when_then_list __
    otherwise:case_else? __
    KW_END __ KW_CASE? {
      if (otherwise) condition_list.push(otherwise);
      return {
        type: 'case',
        expr: expr,
        args: condition_list
      };
    }

case_when_then_list
  = head:case_when_then __ tail:(__ case_when_then)* {
    return createList(head, tail, 1)
  }

case_when_then
  = KW_WHEN __ condition:or_and_where_expr __ KW_THEN __ result:expr {
    return {
      type: 'when',
      cond: condition,
      result: result
    };
  }

case_else = KW_ELSE __ result:expr {
    return { type: 'else', result: result };
  }

column_ref
  = tbl:column_without_kw col:(__ DOT __ column_without_kw)+ __ cof:(column_offset_expr_list __ (DOT __ column_without_kw)?)? {
      const cols = col.map(c => c[3])
      columnList.add(`select::${tbl}::${cols[0]}`)
      const column = cof
      ? {
          column: {
            expr: {
              type: 'column_ref',
              table: null,
              column: cols[0],
              subFields: cols.slice(1)
            },
            offset: cof && cof[0],
            suffix: cof && cof[2] && `.${cof[2][2]}`,
          }
        }
      : { column: cols[0], subFields: cols.slice(1) }
      return {
        type: 'column_ref',
        table: tbl,
        ...column,
        ...getLocationObject(),
      };
    }
  / col:column {
      columnList.add(`select::null::${col}`);
      return {
        type: 'column_ref',
        table: null,
        column: col,
        ...getLocationObject()
      };
    }

column_list
  = head:column tail:(__ COMMA __ column)* {
      return createList(head, tail);
    }

ident_without_kw_type
  = n:ident_name {
    return { type: 'default', value: n }
  }
  / quoted_ident_type

ident_type
  = name:ident_name !{ return reservedMap[name.toUpperCase()] === true; } {
      return { type: 'default', value: name }
    }
  / quoted_ident_type

ident
  = name:ident_name !{ return reservedMap[`${name}`.toUpperCase()] === true; } {
      return name;
    }
  / name:quoted_ident {
      return name;
    }

alias_ident
  = name:column_name !{
      if (reservedMap[name.toUpperCase()] === true) throw new Error("Error: "+ JSON.stringify(name)+" is a reserved word, can not as alias clause");
      return false
    } {
      return name;
    }
  / name:quoted_ident_type {
      return name;
    }

quoted_ident_type
  = double_quoted_ident / single_quoted_ident / backticks_quoted_ident

quoted_ident
  = v:(double_quoted_ident / single_quoted_ident / backticks_quoted_ident) {
    return v.value
  }

double_quoted_ident
  = '"' chars:[^"]+ '"' {
    return {
      type: 'double_quote_string',
      value: chars.join('')
    }
  }

single_quoted_ident
  = "'" chars:[^']+ "'" {
    return {
      type: 'single_quote_string',
      value: chars.join('')
    }
  }

backticks_quoted_ident
  = "`" chars:[^`]+ "`" {
    return {
      type: 'backticks_quote_string',
      value: chars.join('')
    }
  }

column_without_kw
  = column_name / quoted_ident

ident_without_kw
  = ident_name / quoted_ident

column
  = name:column_name !{ return reservedMap[name.toUpperCase()] === true; } { return name; }
  / quoted_ident

column_name
  =  start:ident_start parts:column_part* { return start + parts.join(''); }

ident_name
  =  start:ident_start parts:ident_part* { return start + parts.join(''); }

ident_start = [A-Za-z_]

ident_part  = [A-Za-z0-9_-]

// to support column name like `cf1:name` in hbase
column_part  = [A-Za-z0-9_:]

param
  = s:(':'/'@') n:ident_name {
      return { type: 'param', value: n, prefix: s };
    }

aggr_func_list
  = head:aggr_func __ as:alias_clause? tail:(__ COMMA __ aggr_func __ alias_clause?)* {
      const el = { type: 'expr_list' };
      el.value = createList(head, tail);
      return el;
  }

aggr_func
  = aggr_fun_count
  / aggr_fun_smma

aggr_fun_smma
  = name:KW_SUM_MAX_MIN_AVG  __ LPAREN __ e:additive_expr __ RPAREN __ bc:over_partition? {
      return {
        type: 'aggr_func',
        name: name,
        args: {
          expr: e
        },
        over: bc,
        ...getLocationObject()
      };
    }

KW_SUM_MAX_MIN_AVG
  = KW_SUM / KW_MAX / KW_MIN / KW_AVG

on_update_current_timestamp
  = KW_ON __ 'UPDATE'i __ kw:KW_CURRENT_TIMESTAMP __ LPAREN __ l:expr_list? __ RPAREN{
    return {
      type: 'on update',
      keyword: kw,
      parentheses: true,
      expr: l
    }
  }
  / KW_ON __ 'UPDATE'i __ kw:KW_CURRENT_TIMESTAMP {
    return {
      type: 'on update',
      keyword: kw,
    }
  }

over_partition
  = KW_OVER __ aws:as_window_specification {
    return {
      type: 'window',
      as_window_specification: aws,
    }
  }
  / KW_OVER __ LPAREN __ bc:partition_by_clause __ l:order_by_clause? __ RPAREN {
    return {
      partitionby: bc,
      orderby: l
    }
  }
  / on_update_current_timestamp

aggr_fun_count
  = name:(KW_COUNT / 'string_agg'i) __ LPAREN __ arg:count_arg __ RPAREN __ bc:over_partition? {
      return {
        type: 'aggr_func',
        name: name,
        args: arg,
        over: bc,
        ...getLocationObject()
      };
    }

count_arg
  = e:star_expr { return { expr: e, ...getLocationObject() }; }
  / d:KW_DISTINCT? __ LPAREN __ c:expr __ RPAREN tail:(__ (KW_AND / KW_OR) __ expr)* __ or:order_by_clause? {
    const len = tail.length
    let result = c
    result.parentheses = true
    for (let i = 0; i < len; ++i) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3])
    }
    return {
      distinct: d,
      expr: result,
      orderby: or,
      ...getLocationObject()
    };
  }
  / d:KW_DISTINCT? __ c:or_and_expr __ or:order_by_clause?  { return { distinct: d, expr: c, orderby: or, ...getLocationObject() }; }

star_expr
  = "*" { return { type: 'star', value: '*' }; }

func_call
  = extract_func
  / any_value_func
  / name:scalar_func __ LPAREN __ l:expr_list? __ RPAREN __ bc:over_partition? {
      return {
        type: 'function',
        name: { name: [{ type: 'default', value: name }] },
        args: l ? l: { type: 'expr_list', value: [] },
        over: bc
      };
    }
  / f:scalar_time_func __ up:on_update_current_timestamp? {
    return {
        type: 'function',
        name: { name: [{ type: 'origin', value: f }] },
        over: up
    }
  }
  / name:proc_func_name __ LPAREN __ l:or_and_where_expr? __ RPAREN __ bc:over_partition? {
    if (l && l.type !== 'expr_list') l = { type: 'expr_list', value: [l] }
      return {
        type: 'function',
        name: name,
        args: l ? l: { type: 'expr_list', value: [] },
        over: bc
      };
    }

proc_func_name
  = dt:ident_without_kw_type tail:(__ DOT __ ident_without_kw_type)* {
      const result = { name: [dt] }
      if (tail !== null) {
        result.schema = dt
        result.name = tail.map(t => t[3])
      }
      return result
    }

scalar_time_func
  = KW_CURRENT_DATE
  / KW_CURRENT_TIME
  / KW_CURRENT_TIMESTAMP
scalar_func
  = scalar_time_func / KW_SESSION_USER

any_value_having
  = KW_HAVING __ i:(KW_MAX / KW_MIN) __ e:or_and_where_expr {
    return {
      prefix: i,
      expr: e
    }
  }

any_value_func
  = 'ANY_VALUE'i __ LPAREN __ e:or_and_where_expr __ h:any_value_having? __ RPAREN __ bc:over_partition? {
    return {
        type: 'any_value',
        args: {
          expr: e,
          having: h
        },
        over: bc
    }
  }

extract_filed
  = f:(
    'YEAR_MONTH'i / 'DAY_HOUR'i / 'DAY_MINUTE'i / 'DAY_SECOND'i / 'DAY_MICROSECOND'i / 'HOUR_MINUTE'i / 'HOUR_SECOND'i/ 'HOUR_MICROSECOND'i / 'MINUTE_SECOND'i / 'MINUTE_MICROSECOND'i / 'SECOND_MICROSECOND'i / 'TIMEZONE_HOUR'i / 'TIMEZONE_MINUTE'i
    / 'CENTURY'i / 'DAYOFWEEK'i / 'DAY'i / 'DATE'i / 'DECADE'i / 'DOW'i / 'DOY'i / 'EPOCH'i / 'HOUR'i / 'ISODOW'i / 'ISOWEEK'i / 'ISOYEAR'i / 'MICROSECONDS'i / 'MILLENNIUM'i / 'MILLISECONDS'i / 'MINUTE'i / 'MONTH'i / 'QUARTER'i / 'SECOND'i / 'TIME'i / 'TIMEZONE'i / 'WEEK'i / 'YEAR'i
  ) {
    return f
  }
extract_func
  = kw:KW_EXTRACT __ LPAREN __ f:extract_filed __ KW_FROM __ t:(KW_TIMESTAMP / KW_INTERVAL / KW_TIME / KW_DATE) __ s:expr __ RPAREN {
    return {
        type: kw.toLowerCase(),
        args: {
          field: f,
          cast_type: t,
          source: s,
        }
    }
  }
  / kw:KW_EXTRACT __ LPAREN __ f:extract_filed __ KW_FROM __ s:expr __ RPAREN {
    return {
        type: kw.toLowerCase(),
        args: {
          field: f,
          source: s,
        }
    }
  }
  / 'DATE_TRUNC'i __  LPAREN __ e:expr __ COMMA __ f:extract_filed __ RPAREN {
    return {
        type: 'function',
        name: { name: [{ type: 'origin', value: 'date_trunc' }]},
        args: { type: 'expr_list', value: [e, { type: 'origin', value: f }] },
        over: null,
      };
  }

cast_keyword
  = KW_CAST / KW_SAFE_CAST
cast_expr
  = c:cast_keyword __ LPAREN __ e:expr __ KW_AS __ t:data_type __ RPAREN {
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: t
    };
  }
  / c:cast_keyword __ LPAREN __ e:expr __ KW_AS __ KW_DECIMAL __ LPAREN __ precision:int __ RPAREN __ RPAREN {
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: {
        dataType: 'DECIMAL(' + precision + ')'
      }
    };
  }
  / c:cast_keyword __ LPAREN __ e:expr __ KW_AS __ KW_DECIMAL __ LPAREN __ precision:int __ COMMA __ scale:int __ RPAREN __ RPAREN {
      return {
        type: 'cast',
        keyword: c.toLowerCase(),
        expr: e,
        symbol: 'as',
        target: {
          dataType: 'DECIMAL(' + precision + ', ' + scale + ')'
        }
      };
    }
  / c:cast_keyword __ LPAREN __ e:expr __ KW_AS __ s:signedness __ t:KW_INTEGER? __ RPAREN { /* MySQL cast to un-/signed integer */
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: {
        dataType: s + (t ? ' ' + t: '')
      }
    };
  }

signedness
  = KW_SIGNED
  / KW_UNSIGNED

literal
  = literal_string
  / literal_numeric
  / literal_bool
  / literal_null
  / literal_datetime

literal_list
  = head:literal tail:(__ COMMA __ literal)* {
      return createList(head, tail);
    }

literal_null
  = KW_NULL {
      return { type: 'null', value: null };
    }

literal_not_null
  = KW_NOT_NULL {
    return {
      type: 'not null',
      value: 'not null',
    }
  }

literal_bool
  = KW_TRUE {
      return { type: 'bool', value: true };
    }
  / KW_FALSE {
      return { type: 'bool', value: false };
    }

literal_string
  = r:'R'i? __ ca:("'" single_char* "'") {
      return {
        type: r ? 'regex_string' : 'single_quote_string',
        value: ca[1].join(''),
        ...getLocationObject()
      };
    }
  / r:'R'i? __ ca:("\"" single_quote_char* "\"") {
      return {
        type: r ? 'regex_string' : 'string',
        value: ca[1].join(''),
        ...getLocationObject()
      };
    }

literal_datetime
  = type:(KW_TIME / KW_DATE / KW_TIMESTAMP / KW_DATETIME) __ ca:("'" single_char* "'") {
      return {
        type: type.toLowerCase(),
        value: ca[1].join('')
      };
    }
  / type:(KW_TIME / KW_DATE / KW_TIMESTAMP / KW_DATETIME) __ ca:("\"" single_quote_char* "\"") {
      return {
        type: type.toLowerCase(),
        value: ca[1].join('')
      };
    }

single_quote_char
  = [^"\\\0-\x1F\x7f]
  / escape_char

single_char
  = [^'\\] // remove \0-\x1F\x7f pnCtrl char [^'\\\0-\x1F\x7f]
  / escape_char

escape_char
  = "\\'"  { return "\\'";  }
  / '\\"'  { return '\\"';  }
  / "\\\\" { return "\\\\"; }
  / "\\/"  { return "\\/";  }
  / "\\b"  { return "\b"; }
  / "\\f"  { return "\f"; }
  / "\\n"  { return "\n"; }
  / "\\r"  { return "\r"; }
  / "\\t"  { return "\t"; }
  / "\\u" h1:hexDigit h2:hexDigit h3:hexDigit h4:hexDigit {
      return String.fromCharCode(parseInt("0x" + h1 + h2 + h3 + h4));
    }
  / "\\" { return "\\"; }
  / "''" { return "''" }
  / '""' { return '""' }
  / '``' { return '``' }

line_terminator
  = [\n\r]

literal_numeric
  = n:number {
      if (n && n.type === 'bigint') return n
      return { type: 'number', value: n };
    }

number
  = int_:int frac:frac exp:exp {
    const numStr = int_ + frac + exp
    return {
      type: 'bigint',
      value: numStr
    }
  }
  / int_:int frac:frac {
    const numStr = int_ + frac
    if (isBigInt(int_)) return {
      type: 'bigint',
      value: numStr
    }
    return parseFloat(numStr);
  }
  / int_:int exp:exp {
    const numStr = int_ + exp
    return {
      type: 'bigint',
      value: numStr
    }
  }
  / int_:int {
    if (isBigInt(int_)) return {
      type: 'bigint',
      value: int_
    }
    return parseFloat(int_);
  }

int
  = digits
  / digit:digit
  / op:("-" / "+" ) digits:digits { return op + digits; }
   / op:("-" / "+" ) digit:digit { return op + digit; }

frac
  = "." digits:digits { return "." + digits; }

exp
  = e:e digits:digits { return e + digits; }

digits
  = digits:digit+ { return digits.join(""); }

digit   = [0-9]

hexDigit
  = [0-9a-fA-F]

e
  = e:[eE] sign:[+-]? { return e + (sign !== null ? sign: ''); }


KW_NULL     = "NULL"i       !ident_start
KW_DEFAULT  = "DEFAULT"i    !ident_start
KW_NOT_NULL = "NOT NULL"i   !ident_start
KW_TRUE     = "TRUE"i       !ident_start
KW_TO       = "TO"i         !ident_start
KW_FALSE    = "FALSE"i      !ident_start

KW_DROP     = "DROP"i       !ident_start { return 'DROP'; }
KW_USE      = "USE"i        !ident_start
KW_SELECT   = "SELECT"i     !ident_start
KW_RECURSIVE= "RECURSIVE"   !ident_start
KW_IGNORE   = "IGNORE"i     !ident_start
KW_EXPLAIN  = "EXPLAIN"i    !ident_start
KW_PARTITION = "PARTITION"i !ident_start { return 'PARTITION' }

KW_INTO     = "INTO"i       !ident_start
KW_FROM     = "FROM"i       !ident_start
KW_SET      = "SET"i        !ident_start { return 'SET' }
KW_UNLOCK   = "UNLOCK"i     !ident_start
KW_LOCK     = "LOCK"i       !ident_start

KW_AS       = "AS"i         !ident_start
KW_TABLE    = "TABLE"i      !ident_start { return 'TABLE'; }
KW_TABLES   = "TABLES"i      !ident_start { return 'TABLES'; }
KW_COLLATE  = "COLLATE"i    !ident_start { return 'COLLATE'; }

KW_ON       = "ON"i       !ident_start
KW_LEFT     = "LEFT"i     !ident_start
KW_RIGHT    = "RIGHT"i    !ident_start
KW_FULL     = "FULL"i     !ident_start
KW_INNER    = "INNER"i    !ident_start
KW_CROSS    = "CROSS"i    !ident_start
KW_JOIN     = "JOIN"i     !ident_start
KW_OUTER    = "OUTER"i    !ident_start
KW_OVER     = "OVER"i     !ident_start
KW_UNION    = "UNION"i    !ident_start
KW_INTERSECT    = "INTERSECT"i    !ident_start
KW_EXCEPT    = "EXCEPT"i    !ident_start

KW_VALUE    = "VALUE"i    !ident_start { return 'VALUE' }
KW_VALUES   = "VALUES"i   !ident_start
KW_USING    = "USING"i    !ident_start

KW_WHERE    = "WHERE"i      !ident_start
KW_WITH     = "WITH"i       !ident_start

KW_GROUP    = "GROUP"i      !ident_start
KW_BY       = "BY"i         !ident_start
KW_ORDER    = "ORDER"i      !ident_start
KW_HAVING   = "HAVING"i     !ident_start
KW_QUALIFY  = "QUALIFY"i     !ident_start
KW_WINDOW   = "WINDOW"i  !ident_start
KW_ORDINAL  = "ORDINAL"i !ident_start { return 'ORDINAL' }
KW_SAFE_ORDINAL  = "SAFE_ORDINAL"i !ident_start { return 'SAFE_ORDINAL' }

KW_LIMIT    = "LIMIT"i      !ident_start
KW_OFFSET   = "OFFSET"i     !ident_start { return 'OFFSET'; }
KW_SAFE_OFFSET   = "SAFE_OFFSET"i     !ident_start { return 'SAFE_OFFSET'; }

KW_ASC      = "ASC"i        !ident_start { return 'ASC'; }
KW_DESC     = "DESC"i       !ident_start { return 'DESC'; }

KW_ALL      = "ALL"i        !ident_start { return 'ALL'; }
KW_DISTINCT = "DISTINCT"i   !ident_start { return 'DISTINCT';}

KW_BETWEEN  = "BETWEEN"i    !ident_start { return 'BETWEEN'; }
KW_IN       = "IN"i         !ident_start { return 'IN'; }
KW_IS       = "IS"i         !ident_start { return 'IS'; }
KW_LIKE     = "LIKE"i       !ident_start { return 'LIKE'; }
KW_EXISTS   = "EXISTS"i     !ident_start { return 'EXISTS'; }

KW_NOT      = "NOT"i        !ident_start { return 'NOT'; }
KW_AND      = "AND"i        !ident_start { return 'AND'; }
KW_OR       = "OR"i         !ident_start { return 'OR'; }

KW_COUNT    = "COUNT"i      !ident_start { return 'COUNT'; }
KW_MAX      = "MAX"i        !ident_start { return 'MAX'; }
KW_MIN      = "MIN"i        !ident_start { return 'MIN'; }
KW_SUM      = "SUM"i        !ident_start { return 'SUM'; }
KW_AVG      = "AVG"i        !ident_start { return 'AVG'; }

KW_EXTRACT  = "EXTRACT"i    !ident_start { return 'EXTRACT'; }
KW_CALL     = "CALL"i       !ident_start { return 'CALL'; }

KW_CASE     = "CASE"i       !ident_start
KW_WHEN     = "WHEN"i       !ident_start
KW_THEN     = "THEN"i       !ident_start
KW_ELSE     = "ELSE"i       !ident_start
KW_END      = "END"i        !ident_start

KW_CAST     = "CAST"i       !ident_start { return 'CAST' }
KW_SAFE_CAST     = "SAFE_CAST"i   !ident_start { return 'SAFE_CAST' }

KW_ARRAY     = "ARRAY"i     !ident_start { return 'ARRAY'; }
KW_BYTES     = "BYTES"i     !ident_start { return 'BYTES'; }
KW_BOOL     = "BOOL"i     !ident_start { return 'BOOL'; }
KW_CHAR     = "CHAR"i     !ident_start { return 'CHAR'; }
KW_GEOGRAPHY = "GEOGRAPHY"i     !ident_start { return 'GEOGRAPHY'; }
KW_VARCHAR  = "VARCHAR"i  !ident_start { return 'VARCHAR';}
KW_NUMERIC  = "NUMERIC"i  !ident_start { return 'NUMERIC'; }
KW_DECIMAL  = "DECIMAL"i  !ident_start { return 'DECIMAL'; }
KW_SIGNED   = "SIGNED"i   !ident_start { return 'SIGNED'; }
KW_UNSIGNED = "UNSIGNED"i !ident_start { return 'UNSIGNED'; }
KW_INT_64     = "INT64"i      !ident_start { return 'INT64'; }
KW_ZEROFILL = "ZEROFILL"i !ident_start { return 'ZEROFILL'; }
KW_INTEGER  = "INTEGER"i  !ident_start { return 'INTEGER'; }
KW_JSON     = "JSON"i     !ident_start { return 'JSON'; }
KW_SMALLINT = "SMALLINT"i !ident_start { return 'SMALLINT'; }
KW_STRING = "STRING"i !ident_start { return 'STRING'; }
KW_STRUCT = "STRUCT"i !ident_start { return 'STRUCT'; }
KW_TINYINT  = "TINYINT"i  !ident_start { return 'TINYINT'; }
KW_TINYTEXT = "TINYTEXT"i !ident_start { return 'TINYTEXT'; }
KW_TEXT     = "TEXT"i     !ident_start { return 'TEXT'; }
KW_MEDIUMTEXT = "MEDIUMTEXT"i  !ident_start { return 'MEDIUMTEXT'; }
KW_LONGTEXT  = "LONGTEXT"i  !ident_start { return 'LONGTEXT'; }
KW_BIGINT   = "BIGINT"i   !ident_start { return 'BIGINT'; }
KW_FLOAT_64   = "FLOAT64"i   !ident_start { return 'FLOAT64'; }
KW_DOUBLE   = "DOUBLE"i   !ident_start { return 'DOUBLE'; }
KW_DATE     = "DATE"i     !ident_start { return 'DATE'; }
KW_DATETIME = "DATETIME"i     !ident_start { return 'DATETIME'; }
KW_ROWS     = "ROWS"i     !ident_start { return 'ROWS'; }
KW_TIME     = "TIME"i     !ident_start { return 'TIME'; }
KW_TIMESTAMP= "TIMESTAMP"i!ident_start { return 'TIMESTAMP'; }
KW_TRUNCATE = "TRUNCATE"i !ident_start { return 'TRUNCATE'; }
KW_USER     = "USER"i     !ident_start { return 'USER'; }

KW_CURRENT_DATE     = "CURRENT_DATE"i !ident_start { return 'CURRENT_DATE'; }
KW_ADD_DATE         = "ADDDATE"i !ident_start { return 'ADDDATE'; }
KW_INTERVAL         = "INTERVAL"i !ident_start { return 'INTERVAL'; }
KW_UNIT_YEAR        = "YEAR"i !ident_start { return 'YEAR'; }
KW_UNIT_ISOYEAR     = "ISOYEAR"i !ident_start { return 'ISOYEAR'; }
KW_UNIT_MONTH       = "MONTH"i !ident_start { return 'MONTH'; }
KW_UNIT_DAY         = "DAY"i !ident_start { return 'DAY'; }
KW_UNIT_HOUR        = "HOUR"i !ident_start { return 'HOUR'; }
KW_UNIT_MINUTE      = "MINUTE"i !ident_start { return 'MINUTE'; }
KW_UNIT_SECOND      = "SECOND"i !ident_start { return 'SECOND'; }
KW_UNIT_WEEK        = "WEEK"i !ident_start { return 'WEEK'; }
KW_CURRENT_TIME     = "CURRENT_TIME"i !ident_start { return 'CURRENT_TIME'; }
KW_CURRENT_TIMESTAMP= "CURRENT_TIMESTAMP"i !ident_start { return 'CURRENT_TIMESTAMP'; }
KW_SESSION_USER     = "SESSION_USER"i !ident_start { return 'SESSION_USER'; }

KW_GLOBAL         = "GLOBAL"i    !ident_start { return 'GLOBAL'; }
KW_SESSION        = "SESSION"i   !ident_start { return 'SESSION'; }
KW_LOCAL          = "LOCAL"i     !ident_start { return 'LOCAL'; }
KW_PIVOT          = "PIVOT"i   !ident_start { return 'PIVOT'; }
KW_PERSIST        = "PERSIST"i   !ident_start { return 'PERSIST'; }
KW_PERSIST_ONLY   = "PERSIST_ONLY"i   !ident_start { return 'PERSIST_ONLY'; }
KW_VIEW           = "VIEW"i    !ident_start { return 'VIEW'; }

// MySQL Alter
KW_ADD     = "ADD"i     !ident_start { return 'ADD'; }
KW_COLUMN  = "COLUMN"i  !ident_start { return 'COLUMN'; }
KW_INDEX   = "INDEX"i  !ident_start { return 'INDEX'; }
KW_KEY     = "KEY"i  !ident_start { return 'KEY'; }
KW_FULLTEXT = "FULLTEXT"i  !ident_start { return 'FULLTEXT'; }
KW_UNIQUE     = "UNIQUE"i  !ident_start { return 'UNIQUE'; }
KW_COMMENT     = "COMMENT"i  !ident_start { return 'COMMENT'; }
KW_CONSTRAINT  = "CONSTRAINT"i  !ident_start { return 'CONSTRAINT'; }
KW_REFERENCES  = "REFERENCES"i  !ident_start { return 'REFERENCES'; }

//special character
DOT       = '.'
COMMA     = ','
STAR      = '*'
LPAREN    = '('
RPAREN    = ')'
LANGLE    = '<'
RANGLE    = '>'
LBRAKE    = '['
RBRAKE    = ']'

SEMICOLON = ';'

OPERATOR_CONCATENATION = '||'
OPERATOR_AND = '&&'
LOGIC_OPERATOR = OPERATOR_CONCATENATION / OPERATOR_AND

// separator
__
  = (whitespace / comment)*

___
  = (whitespace / comment)+

comment
  = block_comment
  / line_comment
  / pound_sign_comment

block_comment
  = "/*" (!"*/" char)* "*/"

line_comment
  = "--" (!EOL char)*

pound_sign_comment
  = "#" (!EOL char)*

char = .

interval_unit
  = KW_UNIT_YEAR
  / KW_UNIT_ISOYEAR
  / KW_UNIT_MONTH
  / KW_UNIT_DAY
  / KW_UNIT_HOUR
  / KW_UNIT_MINUTE
  / KW_UNIT_SECOND
  / KW_UNIT_WEEK

whitespace =
  [ \t\n\r]

EOL
  = EOF
  / [\n\r]+

EOF = !.

data_type_list
  = head:data_type_alias tail:(__ COMMA __ data_type_alias)* {
      return createList(head, tail);
    }

data_type_alias
  = n:(n:ident_name !{ return DATA_TYPES[n.toUpperCase()] === true; } {
      return n
    })? __ t:data_type {
    return {
      field_name: n,
      field_type: t,
    }
  }

data_type
  = struct_type
  / array_type
  / character_string_type
  / numeric_type
  / datetime_type
  / bool_byte_geography_type

character_string_type
  = t:KW_STRING s:(__ LPAREN __ l:[0-9]+ __ RPAREN)* {
    const result = { dataType: t }
    if (!s || s.length === 0) return result
    return { ...result, length: parseInt(s[3].join(''), 10), parentheses: true  };
  }

numeric_type
  = t:(KW_NUMERIC / KW_INT_64 / KW_FLOAT_64 / KW_INTEGER) { return { dataType: t }; }

datetime_type
  = t:(KW_DATE / KW_DATETIME / KW_TIME / KW_TIMESTAMP) __ LPAREN __ l:[0-9]+ __ RPAREN { return { dataType: t, length: parseInt(l.join(''), 10), parentheses: true }; }
  / t:(KW_DATE / KW_DATETIME / KW_TIME / KW_TIMESTAMP) { return { dataType: t }; }

bool_byte_geography_type
  = t:( ( KW_BYTES LPAREN __ l:([0-9]+ / "MAX" / "max" ) __ RPAREN ) / KW_BOOL / KW_GEOGRAPHY) { return { dataType: t }; }

array_type
  = t:KW_ARRAY __ LANGLE __ a:data_type_list __ RANGLE {
    return {
      dataType: t,
      definition: a,
      anglebracket: true
    }
  }

struct_type
  = t:KW_STRUCT __ LANGLE __ a:data_type_list __ RANGLE {
    return {
      dataType: t,
      definition: a,
      anglebracket: true
    }
  }
