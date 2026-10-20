import { chromium } from 'playwright';
import assert from 'node:assert/strict';
// Run against a running frontend; API calls are intercepted by Playwright.
const base = process.env.MONITOR_TEST_URL || 'http://localhost:3100';
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(base).hostname), 'Use a local frontend');
const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_EXECUTABLE ? { executablePath: process.env.BROWSER_EXECUTABLE } : {}) });
const page = await browser.newPage();
page.setDefaultTimeout(10000);
let monitors = Array.from({length:23},(_,i)=>({id:String(i),url:`https://example.com/${i}`,interval_seconds:i===0?61:60,created_at:'2026-10-20T00:00:00Z'}));
let failUpdate=true, refreshUpdate=true, failDelete=true, refreshDelete=true, failList=false;
const calls=[];
await page.route('**/api/v1/**', async route=>{
 const req=route.request(), path=new URL(req.url()).pathname, method=req.method(); calls.push({path,method,body:req.postData()});
 const json=(status,data)=>route.fulfill({status,json:data});
 if(path.endsWith('/auth/refresh')) return json(200,{access_token:'token',user:{id:'owner',email:'test@example.com',email_verified:true,full_name:'Test',avatar_url:''}});
 if(path.endsWith('/health')) return json(200,{status:'ok'});
 if(path==='/api/v1/monitors') {
  if(failList){failList=false;return json(503,{error:{message:'List failed'}});}
  const before=new URL(req.url()).searchParams.get('before'); const index=before===null?0:monitors.findIndex(m=>m.id===before)+1;
  return json(200,monitors.slice(index,index+20));
 }
 const id=path.split('/').at(-1), index=monitors.findIndex(m=>m.id===id);
 if(method==='PUT') {
  if(failUpdate){failUpdate=false;return json(503,{error:{message:'Save failed'}});}
  if(refreshUpdate){refreshUpdate=false;return json(401,{error:{message:'Refresh'}});}
  monitors[index]={...monitors[index],...req.postDataJSON()}; return json(200,monitors[index]);
 }
 if(method==='DELETE'){
  if(failDelete){failDelete=false;return json(503,{error:{message:'Delete failed'}});}
  if(refreshDelete){refreshDelete=false;return json(401,{error:{message:'Refresh'}});}
  monitors.splice(index,1);return route.fulfill({status:204});
 }
 return json(404,{});
});
const button=name=>page.getByRole('button',{name,exact:true});
const confirmation=page.getByRole('region',{name:'Подтверждение удаления'});
try {
 await page.goto(base);
 await button('Редактировать https://example.com/0').click();
 assert.equal(await page.locator('#monitor-unit').inputValue(),'1');
 assert.equal(await page.getByLabel('Проверять каждые',{exact:true}).inputValue(),'61');
 await button('Отмена').click();
 await button('Редактировать https://example.com/0').click();
 await page.getByLabel('URL сайта',{exact:true}).fill('https://edited.example.com/');
 await button('Сохранить').click();await page.getByRole('alert').filter({hasText:'Save failed'}).waitFor();
 assert.equal(await page.getByLabel('URL сайта',{exact:true}).inputValue(),'https://edited.example.com/');
 await button('Сохранить').click();await page.getByText('Изменения сохранены.',{exact:true}).waitFor();
 await page.reload();await button('Удалить https://edited.example.com/').click();
 await confirmation.getByRole('button',{name:'Отмена',exact:true}).click();
 assert.equal(monitors.length,23);
 await button('Удалить https://example.com/19').click();
 await confirmation.getByRole('button',{name:'Удалить',exact:true}).click();await page.getByText('Delete failed',{exact:true}).waitFor();
 await confirmation.getByRole('button',{name:'Удалить',exact:true}).click();await page.getByText('Сайт удалён.',{exact:true}).waitFor();
 await button('Загрузить ещё').click();await page.waitForFunction(()=>document.querySelectorAll('.monitor-list li').length===22);
 assert.equal(calls.filter(c=>c.method==='PUT').length,3);assert.equal(calls.filter(c=>c.method==='DELETE').length,3);
 assert.equal(calls.filter(c=>c.method==='PUT').at(-1).body,calls.filter(c=>c.method==='PUT').at(-2).body);
 monitors=[monitors[0]];await page.reload();await button('Удалить https://edited.example.com/').click();
 failList=true;
 await confirmation.getByRole('button',{name:'Удалить',exact:true}).click();
 await page.getByText('Сайт удалён.',{exact:true}).waitFor();await page.getByText('List failed',{exact:true}).waitFor();
 await button('Повторить загрузку').click();await page.getByText('Сайтов пока нет.',{exact:false}).waitFor();
 assert.equal(monitors.length,0);
 console.log('PASS UI edit, exact interval, cancel, errors, refreshed PUT/DELETE, 204, deleted pagination cursor, last site, separate reload failure');

}finally{await browser.close();}
