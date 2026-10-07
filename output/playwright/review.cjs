async page => {
  const base = 'http://127.0.0.1:4173/design-prototypes/';
  const schemes = ['editorial', 'minimal', 'midnight', 'garden'];
  const errors = [];
  const checks = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if(message.type() === 'error') errors.push(message.text()); });
  const ensure = (test, message) => { if(!test) throw new Error(message); };
  for (const scheme of schemes) {
    for (const [kind, path] of [['home', 'index.html'], ['article', 'a-place-for-thoughts.html']]) {
      for (const width of [1440, 768, 390, 320]) {
        await page.setViewportSize({width, height:width === 1440 ? 1040 : 844});
        await page.goto(base + scheme + '/' + path);
        await page.evaluate(() => document.fonts.ready);
        const measure = await page.evaluate(() => ({width:innerWidth, content:document.documentElement.scrollWidth, h1:document.querySelectorAll('h1').length, text:document.querySelector('h1').innerText}));
        ensure(measure.content <= width, `${scheme}/${kind}: horizontal overflow at ${width}: ${measure.content}`);
        ensure(measure.h1 === 1, `${scheme}/${kind}: h1 count ${measure.h1}`);
        if(width === 1440 || width === 390) await page.screenshot({path:`output/playwright/${scheme}-${kind === 'article' ? 'article-' : ''}${width===1440?'desktop':'mobile'}.png`,fullPage:width===390});
        checks.push(`${scheme} ${kind} ${width}px: no overflow`);
      }
    }
    await page.setViewportSize({width:1440,height:1040});
    await page.goto(base+scheme+'/index.html');
    await page.locator('[data-filter="技术"]').click();
    ensure(await page.locator('.post-row:visible').count()===2,scheme+': category filter');
    await page.locator('[data-filter="全部"]').click();
    ensure(await page.locator('.post-row:visible').count()===5,scheme+': reset filter');
    await page.locator('[data-tag="工作方法"]').first().click();
    ensure(await page.locator('.post-row:visible').count()===2,scheme+': tag filter');
    await page.locator('[data-filter="全部"]').click();
    await page.keyboard.press('/');
    await page.locator('#search-input').fill('收藏夹');
    ensure(await page.locator('.search-result').count()===1,scheme+': body full-text search');
    ensure((await page.locator('.search-result strong').innerText()).includes('慢慢生长'),scheme+': correct search result');
    await page.locator('#search-input').fill('根本不存在的关键词');
    ensure(await page.locator('.search-result').count()===0,scheme+': empty search results');
    await page.locator('#search-input').fill('工具');
    ensure(await page.locator('.search-result').count()>1,scheme+': multi-result search');
    await page.keyboard.press('Escape');
    ensure(await page.locator('dialog[open]').count()===0,scheme+': escape closes dialog');
    await page.locator('[data-dialog="categories"]').click();
    await page.locator('#categories-dialog a').filter({hasText:'阅读'}).click();
    ensure(await page.locator('.post-row:visible').count()===1,scheme+': category navigation');
    await page.locator('[data-dialog="tags"]').click();
    await page.locator('#tags-dialog a').filter({hasText:'# AI'}).click();
    ensure(await page.locator('.post-row:visible').count()===1,scheme+': tag navigation');
    await page.locator('[data-dialog="about"]').first().click();
    ensure(await page.locator('#about-dialog').isVisible(),scheme+': about dialog');
    await page.keyboard.press('Escape');
    await page.goto(base+scheme+'/a-place-for-thoughts.html');
    await page.locator('.toc a[href="#section-2"]').click();
    ensure(page.url().endsWith('#section-2'),scheme+': toc navigation');
    await page.locator('.prototype-bar nav a').nth((schemes.indexOf(scheme)+1)%4).click();
    ensure(page.url().includes('/a-place-for-thoughts.html'),scheme+': scheme switch preserves article');
    await page.setViewportSize({width:390,height:844});
    await page.goto(base+scheme+'/index.html');
    await page.locator('[data-dialog="search"]').click();
    await page.locator('#search-input').fill('阅读');
    ensure(await page.locator('.search-result').count()>0,scheme+': mobile search');
    ensure(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),scheme+': mobile dialog overflow');
    await page.keyboard.press('Escape');
    checks.push(scheme+': category, tag, body search, empty state, keyboard, dialogs, article TOC, scheme switching and mobile search passed');
  }
  await page.setViewportSize({width:1440,height:1040});
  await page.goto(base+'index.html');
  ensure(await page.locator('.proposal').count()===4,'Gallery: four proposals');
  ensure(await page.locator('img').evaluateAll(images=>images.every(i=>i.complete&&i.naturalWidth>0)),'Gallery: loaded screenshots');
  await page.screenshot({path:'output/playwright/gallery-desktop.png',fullPage:true});
  await page.setViewportSize({width:390,height:844});
  ensure(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'Gallery: mobile overflow');
  await page.screenshot({path:'output/playwright/gallery-mobile.png',fullPage:true});
  ensure(errors.length===0,'Browser errors: '+errors.join('; '));
  console.log(JSON.stringify({status:'passed',checks,consoleErrors:errors},null,2));
}
