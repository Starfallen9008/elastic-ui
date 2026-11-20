(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (ok,message) => { if (!ok) throw new Error(message); };
  const until = async fn => {for(let i=0;i<200;i++){if(fn())return;await pause(20);}throw new Error('Timed out');};
  const button = label => [...document.querySelectorAll('button')].find(el=>el.textContent.trim()===label);
  const bridge = window.go.main.App;
  await until(()=>document.querySelector('.position-settings'));
  let independent=false, name='Shared policy', writes=0,release;
  bridge.GetIndependentPolicyState=async()=>({independent,applied:independent,name,hostCount:independent?1:2,policyId:independent?'copy':'shared',packageId:'defend',message:independent?'本机独立策略已生效':'当前策略由两台主机共用'});
  bridge.CreateIndependentPolicy=async value=>{writes++;await new Promise(resolve=>release=resolve);independent=true;name=value;return bridge.GetIndependentPolicyState();};
  document.querySelector('.position-settings').click();
  await until(()=>button('创建并绑定本机'));
  const input=document.querySelector('#independent-policy-name');input.value='My independent policy';input.dispatchEvent(new Event('input',{bubbles:true}));
  button('创建并绑定本机').click();await until(()=>release);
  assert(input.disabled,'Name editable during migration');assert(writes===1,'Duplicate writes');release();
  await until(()=>button('保存名称'));
  assert(document.body.textContent.includes('本机独立策略已生效'),'Success hidden');
  bridge.RenameIndependentPolicy=async()=>{throw new Error('Fixture permission denied');};
  button('保存名称').click();await until(()=>document.querySelector('.policy-error'));
  assert(document.querySelector('.policy-error').textContent.includes('Fixture permission denied'),'Failure hidden');
  assert(button('自定义防护策略'),'Custom policy link missing');
  const settings=await bridge.GetSettings();settings.policySetupComplete=false;bridge.GetSettings=async()=>settings;
  window.fixtureEvents.configured();await until(()=>button('进入控制台'));
  let complete=0;bridge.CompletePolicySetup=async()=>{complete++;settings.policySetupComplete=true;};
  button('进入控制台').click();await until(()=>document.querySelector('.position-settings'));
  assert(complete===1,'Onboarding completion not persisted');
  return {passed:true,migrationWrites:writes,onboardingCompleted:complete};
})()
