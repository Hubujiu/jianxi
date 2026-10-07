async page => {
  const base = 'http://127.0.0.1:4173/design-prototypes/';
  await page.setViewportSize({width:1440,height:1040});
  await page.goto(base+'index.html');
  await page.locator('.proposal-grid').screenshot({path:'output/playwright/four-directions.png'});
  const context = await page.context().browser().newContext({javaScriptEnabled:false,reducedMotion:'reduce',viewport:{width:1440,height:1040}});
  const reading = await context.newPage();
  const checks = [];
  for (const scheme of ['editorial','minimal','midnight','garden']) {
    await reading.goto(base+scheme+'/a-place-for-thoughts.html');
    if(await reading.locator('.prose section').count()!==3) throw new Error(scheme+': static article content missing');
    await reading.locator('.toc a[href="#section-2"]').click();
    if(!reading.url().endsWith('#section-2')) throw new Error(scheme+': static anchor failed');
    await reading.locator('.next-article').click();
    if(!reading.url().endsWith('smaller-software.html')) throw new Error(scheme+': static navigation failed');
    checks.push(scheme+': article content, TOC and next article work without JavaScript');
  }
  await context.close();
  return {status:'passed',checks};
}
