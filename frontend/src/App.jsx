import React, { useState, useEffect } from 'react';
import './index.css';

let toastCounter = 0;

function ToastContainer({ toasts }) {
  return (
    <div className="toast-container">
      {toasts.map(t => (
        <div key={t.id} className={`toast ${t.type}`}>
          <div style={{ flex: 1 }}>{t.message}</div>
        </div>
      ))}
    </div>
  );
}

function App() {
  const [token, setToken] = useState(null);
  const [username, setUsername] = useState('');
  const [page, setPage] = useState('dashboard');
  const [role, setRole] = useState('consumer');
  const [userId, setUserId] = useState('');
  
  const [toasts, setToasts] = useState([]);
  const addToast = (msg, type='info') => {
    const id = ++toastCounter;
    setToasts(prev => [...prev, { id, message: msg, type }]);
    setTimeout(() => { setToasts(prev => prev.filter(t => t.id !== id)); }, 4000);
  };

  const handleSetToken = (t, u) => {
    if (!t) {
      setToken(null);
      return;
    }
    setToken(t);
    setUsername(u);
    try {
      let b64 = t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
      const pad = b64.length % 4;
      if (pad) b64 += '='.repeat(4 - pad);
      const payload = JSON.parse(atob(b64));
      const r = payload.role || 'consumer';
      setRole(r);
      setUserId(String(payload.user_id || payload.sub || ''));
      setPage(r === 'admin' ? 'inventory' : 'dashboard');
    } catch(e) { console.error("JWT Decode error:", e); }
  };

  if (!token) {
    return (
      <>
        <Login handleSetToken={handleSetToken} addToast={addToast} />
        <ToastContainer toasts={toasts} />
      </>
    );
  }
  
  return (
    <>
      <MainLayout 
        token={token} username={username} handleSetToken={handleSetToken} 
        page={page} setPage={setPage} addToast={addToast} role={role} userId={userId}
      />
      <ToastContainer toasts={toasts} />
    </>
  );
}

function Login({ handleSetToken, addToast }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [shake, setShake] = useState(false);
  const [loginType, setLoginType] = useState('consumer');
  const [isRegistering, setIsRegistering] = useState(false);

  const handleLogin = async (e) => {
    e.preventDefault();
    if (!username || !password) {
      addToast('Please fill in both fields', 'error');
      triggerShake();
      return;
    }

    setLoading(true);
    try {
      const endpoint = isRegistering ? '/api/v1/auth/register' : '/api/v1/auth/login';
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || (isRegistering ? 'Registration failed' : 'Login failed'));
      
      if (isRegistering) {
        addToast(`Account created successfully! Please login.`, 'success');
        setIsRegistering(false);
        setPassword('');
      } else if (data.token) {
        handleSetToken(data.token, username);
        addToast(`Welcome back, ${username}!`, 'success');
      }
    } catch (err) {
      addToast(err.message, 'error');
      triggerShake();
    } finally {
      setLoading(false);
    }
  };

  const triggerShake = () => {
    setShake(true);
    setTimeout(() => setShake(false), 400);
  };

  return (
    <div className={`glass-panel animate-fade ${shake ? 'shake' : ''}`} style={{ maxWidth: '450px', width: '100%' }}>
      <h1 className="title">K8s Core</h1>
      <p className="subtitle">{loginType === 'admin' ? 'Administrative Access Gateway' : 'Secure Consumer Marketplace'}</p>

      <div style={{ display: 'flex', gap: '1rem', marginBottom: '1.5rem', justifyContent: 'center' }}>
        <button type="button" className={`btn-outline ${loginType === 'consumer' ? 'active' : ''}`} onClick={() => {setLoginType('consumer'); setUsername('');}} style={{ flex: 1, borderColor: loginType === 'consumer' ? 'var(--primary-color)' : '', background: loginType === 'consumer' ? 'rgba(102, 252, 241, 0.1)' : 'transparent' }}>Consumer Portal</button>
        <button type="button" className={`btn-outline ${loginType === 'admin' ? 'active' : ''}`} onClick={() => {setLoginType('admin'); setUsername('admin');}} style={{ flex: 1, borderColor: loginType === 'admin' ? 'var(--error)' : '', color: loginType === 'admin' ? 'var(--error)' : '', background: loginType === 'admin' ? 'rgba(255, 0, 0, 0.1)' : 'transparent' }}>Admin Portal</button>
      </div>

      <form onSubmit={handleLogin}>
        <div className="form-group">
          <label>{loginType === 'admin' ? 'Admin Username' : 'Consumer Username'}</label>
          <input type="text" placeholder={loginType === 'admin' ? 'admin' : 'e.g. jdoe'} value={username} onChange={e => setUsername(e.target.value)} disabled={loginType === 'admin'} />
        </div>
        <div className="form-group">
          <label>Password</label>
          <input type="password" placeholder="••••••••" value={password} onChange={e => setPassword(e.target.value)} />
        </div>
        <button type="submit" disabled={loading} style={{ background: loginType === 'admin' ? 'var(--error)' : '', color: loginType === 'admin' ? '#fff' : '' }}>
          {loading ? 'Authenticating...' : isRegistering ? 'Create Secure Account' : `Login to ${loginType === 'admin' ? 'Admin' : 'Consumer'} Dashboard`}
        </button>
        {loginType === 'consumer' && (
            <div style={{ textAlign: 'center', marginTop: '1rem' }}>
                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', cursor: 'pointer', textDecoration: 'underline' }} onClick={() => setIsRegistering(!isRegistering)}>
                    {isRegistering ? 'Already have an account? Login here' : 'Need an account? Register digitally'}
                </span>
            </div>
        )}
      </form>
    </div>
  );
}

function MainLayout({ token, username, handleSetToken, page, setPage, addToast, role, userId }) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [notifications, setNotifications] = useState([]);
  const [readCount, setReadCount] = useState(0);

  useEffect(() => {
    const fetchNotifs = async () => {
      try {
        const res = await fetch('/api/v1/notifications');
        if (res.ok) {
          const data = await res.json();
          let parsed = Array.isArray(data) ? data : (data.notifications || []);
          parsed = parsed.map((obj, i) => {
            let p = typeof obj === 'string' ? JSON.parse(obj) : obj;
            p._id = i;
            return p;
          });
          
          parsed = parsed.filter(n => {
            if (role === 'admin') return true; 
            if (role === 'consumer' && n.type === 'ORDER_PLACED' && String(n.user_id) === String(userId)) return true;
            return false;
          });
          
          setNotifications(parsed);
        }
      } catch (e) {
        console.error("Failed to fetch notifications", e);
      }
    };
    fetchNotifs();
    const intv = setInterval(fetchNotifs, 3000);
    return () => clearInterval(intv);
  }, [role, userId]);

  const unreadAmount = Math.max(0, notifications.length - readCount);

  return (
    <div style={{ width: '100%' }}>
      <nav className="navbar animate-fade">
        <h2 className="title" style={{ fontSize: '1.8rem', margin: 0 }}>
          {role === 'admin' ? 'Admin Gateway' : 'K8s Market'}
        </h2>
        <div className="nav-links">
          {role === 'consumer' && <span className={`nav-item ${page === 'dashboard' ? 'active' : ''}`} onClick={() => setPage('dashboard')}>Dashboard</span>}
          {role === 'consumer' && <span className={`nav-item ${page === 'orders' ? 'active' : ''}`} onClick={() => setPage('orders')}>Orders</span>}
          {role === 'admin' && <span className={`nav-item ${page === 'inventory' ? 'active' : ''}`} onClick={() => setPage('inventory')}>Inventory</span>}
          <span className={`nav-item ${page === 'profile' ? 'active' : ''}`} onClick={() => setPage('profile')}>Profile</span>
        </div>
        <div style={{ display: 'flex', gap: '1.5rem', alignItems: 'center' }}>
          <div className="bell-icon" onClick={() => setDrawerOpen(true)} style={{ position: 'relative', cursor: 'pointer' }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
            {unreadAmount > 0 && <span style={{ position: 'absolute', top: '-5px', right: '-5px', background: 'var(--error)', color: '#fff', fontSize: '0.75rem', fontWeight: 'bold', padding: '2px 6px', borderRadius: '50px' }}>{unreadAmount}</span>}
          </div>
        </div>
      </nav>

      {/* Pages */}
      <div className="animate-fade" key={page}>
        {page === 'dashboard' && role === 'consumer' && <Dashboard token={token} username={username} addToast={addToast} />}
        {page === 'orders' && role === 'consumer' && <Orders token={token} />}
        {page === 'inventory' && role === 'admin' && <Inventory token={token} addToast={addToast} />}
        {page === 'profile' && <Profile token={token} username={username} handleSetToken={handleSetToken} role={role} userId={userId}/>}
      </div>

      {/* Drawer */}
      <div className={`drawer-overlay ${drawerOpen ? 'open' : ''}`} onClick={() => setDrawerOpen(false)}></div>
      <div className={`drawer ${drawerOpen ? 'open' : ''}`}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem' }}>
          <h3 style={{ margin: 0 }}>Live Notifications</h3>
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
            {unreadAmount > 0 && (
              <button className="btn-small btn-outline" onClick={() => setReadCount(notifications.length)} style={{ fontSize: '0.8rem', padding: '0.3rem 0.6rem' }}>Mark all read</button>
            )}
            <button className="btn-small btn-outline" onClick={() => setDrawerOpen(false)} style={{ fontSize: '1.2rem', padding: '0 0.5rem', border: 'none' }} title="Close">✕</button>
          </div>
        </div>
        
        {notifications.length === 0 ? (
          <p style={{ color: 'var(--text-muted)' }}>No recent activity.</p>
        ) : (
          notifications.map((n, i) => (
            <div key={i} className="notification-item" style={{ opacity: i < unreadAmount ? 1 : 0.6 }}>
              {n.type === 'LOW_STOCK' ? (
                <>
                  <strong style={{color: 'var(--error)'}}>Low Stock Alert!</strong><br/>
                  Product <strong>{n.product_id}</strong> is running out (Only {n.remaining} left)
                </>
              ) : n.type === 'ORDER_PLACED' ? (
                <>
                  <strong>Order {n.order_id?.substring(0, 8)}</strong><br/>
                  <span style={{ color: 'var(--primary-color)', fontWeight: 'bold'}}>Placed</span> for Product: {n.product_id}
                  {role === 'admin' && <><br /><span style={{fontSize: '0.8rem', color: 'var(--text-muted)'}}>User ID: {n.user_id}</span></>}
                </>
              ) : (
                <>
                  <strong>Event</strong><br/>
                  {n.message || 'Unknown Status'}
                </>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function Dashboard({ token, username, addToast }) {
  const [loading, setLoading] = useState(false);
  const [lastOrder, setLastOrder] = useState(null);

  const placeOrder = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ product_id: 'laptop-01', quantity: 1 })
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Order failed');
      addToast(`Order ${data.order_id.substring(0,8)} placed successfully!`, 'success');
      setLastOrder({ id: data.order_id, status: data.status, time: new Date().toLocaleTimeString() });
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="glass-panel">
      <h2 style={{ fontSize: '2rem', marginBottom: '0.25rem' }}>Welcome back, {username}!</h2>
      <p className="subtitle" style={{ marginBottom: '2.5rem' }}>Here is your overview for today.</p>

      {lastOrder && (
        <div className="animate-fade" style={{ background: 'rgba(46, 204, 113, 0.1)', border: '1px solid var(--success)', borderRadius: '12px', padding: '1rem', marginBottom: '1.5rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ color: 'var(--success)', fontWeight: 'bold' }}>Live Order Status: {lastOrder.status}</span>
          <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>ID: {lastOrder.id.substring(0,8)} • {lastOrder.time}</span>
        </div>
      )}

      <div className="product-card">
        <div className="product-info">
          <h3 style={{ marginBottom: '0.2rem' }}>MacBook Pro M3 Max</h3>
          <p style={{ color: 'var(--text-muted)', margin: 0, fontSize: '0.9rem' }}>ID: laptop-01</p>
          <div className="badge">Featured Product</div>
        </div>
        <div style={{ width: '200px' }}>
          <button onClick={placeOrder} disabled={loading}>
            {loading ? 'Processing...' : 'Place Order'}
          </button>
        </div>
      </div>
    </div>
  );
}

function Orders({ token }) {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/v1/orders', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => res.json())
      .then(data => {
        setOrders(data.orders || (Array.isArray(data) ? data : []));
        setLoading(false);
      })
      .catch(e => {
        console.error(e);
        setLoading(false);
      });
  }, [token]);

  return (
    <div className="glass-panel">
      <h2 style={{ marginBottom: '1.5rem' }}>Order History</h2>
      {loading ? (
        <div><div className="skeleton skeleton-row"></div><div className="skeleton skeleton-row"></div></div>
      ) : orders.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '3rem 0', color: 'var(--text-muted)' }}>
          <svg width="60" height="60" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.2)" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" style={{ marginBottom: '1rem' }}>
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
            <line x1="8" y1="21" x2="16" y2="21" />
            <line x1="12" y1="17" x2="12" y2="21" />
          </svg>
          <p style={{ fontSize: '1.2rem', marginBottom: '0.5rem', color: 'var(--text-main)' }}>No orders found</p>
          <p style={{ fontSize: '0.9rem' }}>When you place an order, it will appear here.</p>
        </div>
      ) : (
        <table className="premium-table">
          <thead><tr><th>ID</th><th>Product</th><th>Qty</th><th>Status</th></tr></thead>
          <tbody>
            {orders.map((o, index) => (
              <tr key={index}>
                <td style={{ fontFamily: 'monospace' }}>{o.order_id}</td>
                <td>{o.product_id}</td>
                <td>{o.quantity}</td>
                <td><span className="badge">{o.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function Inventory({ token, addToast }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(null);
  const [newProdId, setNewProdId] = useState('');
  const [newQty, setNewQty] = useState('');

  const fetchInv = () => {
    fetch('/api/v1/inventory', { headers: { 'Authorization': `Bearer ${token}` }})
      .then(res => res.json())
      .then(data => { setItems(data.items || (Array.isArray(data) ? data : [])); setLoading(false); })
      .catch(() => setLoading(false));
  };
  useEffect(() => { fetchInv(); }, [token]);

  const saveRow = async (prodId, sqty) => {
    const qty = parseInt(sqty);
    if(isNaN(qty)) return;
    setUpdating(prodId);
    try {
      const res = await fetch(`/api/v1/inventory/${prodId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ quantity: qty })
      });
      if (!res.ok) throw new Error('Update failed');
      addToast(`Updated stock for ${prodId}`, 'success');
      fetchInv(); // refresh
    } catch(e) {
      addToast(e.message, 'error');
    } finally {
      setUpdating(null);
    }
  };

  const addProduct = async (e) => {
      e.preventDefault();
      if (!newProdId || !newQty) return;
      await saveRow(newProdId, newQty);
      setNewProdId(''); setNewQty('');
  };

  return (
    <div className="glass-panel">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
          <h2 style={{ margin: 0 }}>Inventory Management</h2>
          <form onSubmit={addProduct} style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
              <input type="text" placeholder="Product ID" value={newProdId} onChange={(e) => setNewProdId(e.target.value)} className="inline-input" style={{ width: '120px' }} />
              <input type="number" placeholder="Qty" value={newQty} onChange={(e) => setNewQty(e.target.value)} className="inline-input" style={{ width: '60px' }} />
              <button type="submit" className="btn-small">Add / Update</button>
          </form>
      </div>

      {loading ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1.5rem' }}>
          <div className="skeleton" style={{ height: '180px', borderRadius: '16px' }}></div>
          <div className="skeleton" style={{ height: '180px', borderRadius: '16px' }}></div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1.5rem' }}>
          {items.map((i, idx) => (
            <InventoryCard key={idx} item={i} onSave={saveRow} updating={updating === i.product_id} />
          ))}
        </div>
      )}
    </div>
  );
}

function InventoryCard({ item, onSave, updating }) {
  const [qty, setQty] = useState(item.quantity);
  const isLow = qty < 5;
  const isDirty = qty != item.quantity;
  const maxStock = 50; 
  const progressPercent = Math.min(100, (qty / maxStock) * 100);
  
  return (
    <div className={`product-card ${isLow ? 'warning-glow' : ''}`} style={{ flexDirection: 'column', alignItems: 'flex-start', margin: 0, padding: '1.5rem', height: '100%', boxSizing: 'border-box' }}>
      <div style={{ width: '100%', display: 'flex', justifyContent: 'space-between', marginBottom: '1.5rem', alignItems: 'center' }}>
        <h3 style={{ margin: 0, fontSize: '1.2rem', fontFamily: 'monospace' }}>{item.product_id}</h3>
        {isLow && <span style={{ color: 'var(--error)', fontSize: '0.8rem', fontWeight: 'bold', background: 'rgba(255,0,0,0.1)', padding: '4px 8px', borderRadius: '4px' }}>Low Stock</span>}
      </div>
      
      <div style={{ width: '100%', marginBottom: '2rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
          <span>Stock Level</span>
          <span>{qty} Units</span>
        </div>
        <div style={{ height: '10px', background: 'rgba(255,255,255,0.1)', borderRadius: '5px', overflow: 'hidden' }}>
          <div style={{ 
            height: '100%', 
            width: `${progressPercent}%`, 
            background: isLow ? 'var(--error)' : 'linear-gradient(90deg, var(--primary-color), var(--secondary-color))',
            transition: 'width 0.5s ease-out'
          }}></div>
        </div>
      </div>

      <div style={{ display: 'flex', width: '100%', gap: '1rem', marginTop: 'auto' }}>
        <input 
          type="number" 
          value={qty} 
          onChange={e => setQty(e.target.value)} 
          className="inline-input" 
          style={{ flex: 1, textAlign: 'left', padding: '0.8rem' }}
        />
        <button 
          className="btn-small btn-outline" 
          disabled={!isDirty || updating}
          onClick={() => onSave(item.product_id, qty)}
          style={{ width: '80px', padding: '0.8rem' }}
        >
          {updating ? '...' : 'Save'}
        </button>
      </div>
    </div>
  );
}

function Profile({ username, handleSetToken, role, userId, token }) {
  const [totalOrders, setTotalOrders] = useState(0);

  useEffect(() => {
    if (role === 'consumer') {
        fetch('/api/v1/orders', { headers: { 'Authorization': `Bearer ${token}` }})
        .then(res => res.json())
        .then(data => { setTotalOrders(data ? data.length : 0); })
        .catch(e => console.error(e));
    }
  }, [token, role]);

  return (
    <div className="glass-panel animate-fade" style={{ maxWidth: '400px', margin: '0 auto', textAlign: 'center', padding: '3rem' }}>
      <div style={{ width: '90px', height: '90px', borderRadius: '50%', background: 'linear-gradient(135deg, var(--primary-color), var(--secondary-color))', margin: '0 auto 1.5rem', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '2.5rem', color: '#0b0c10', fontWeight: 'bold', boxShadow: '0 0 20px rgba(102, 252, 241, 0.4)' }}>
        {username.charAt(0).toUpperCase()}
      </div>
      <h2 style={{ fontSize: '1.8rem', textTransform: 'capitalize' }}>{username}</h2>
      <p style={{ color: 'var(--text-muted)', marginBottom: '2.5rem' }}>User ID: {userId}</p>
      
      <div style={{ background: 'rgba(0,0,0,0.2)', padding: '1.5rem', borderRadius: '16px', marginBottom: '2.5rem', border: '1px solid var(--panel-border)' }}>
        <h4 style={{ margin: '0 0 1rem 0', color: 'var(--text-main)', fontSize: '1.1rem' }}>Account Profile</h4>
        <div style={{ display: 'flex', justifyContent: 'space-around', color: 'var(--text-muted)' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '5px' }}>
            <span style={{ fontSize: '1.5rem', color: 'var(--primary-color)', fontWeight: 'bold', textTransform: 'uppercase' }}>{role}</span>
            <span style={{ fontSize: '0.85rem', textTransform: 'uppercase', letterSpacing: '1px' }}>Role Level</span>
          </div>
          {role === 'consumer' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '5px' }}>
                <span style={{ fontSize: '1.5rem', color: 'var(--primary-color)', fontWeight: 'bold' }}>{totalOrders}</span>
                <span style={{ fontSize: '0.85rem', textTransform: 'uppercase', letterSpacing: '1px' }}>Total Orders</span>
              </div>
          )}
        </div>
      </div>
      
      <button className="btn-outline" onClick={() => handleSetToken(null)} style={{ borderColor: 'var(--error)', color: 'var(--error)', width: '100%', padding: '1rem', fontWeight: 'bold' }}>
        Logout Securely
      </button>
    </div>
  );
}

export default App;
