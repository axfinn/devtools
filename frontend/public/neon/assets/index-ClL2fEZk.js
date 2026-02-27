var Uv=Object.defineProperty;var zv=(r,t,n)=>t in r?Uv(r,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):r[t]=n;var ve=(r,t,n)=>zv(r,typeof t!="symbol"?t+"":t,n);function Hv(r,t){for(var n=0;n<t.length;n++){const a=t[n];if(typeof a!="string"&&!Array.isArray(a)){for(const i in a)if(i!=="default"&&!(i in r)){const s=Object.getOwnPropertyDescriptor(a,i);s&&Object.defineProperty(r,i,s.get?s:{enumerable:!0,get:()=>a[i]})}}}return Object.freeze(Object.defineProperty(r,Symbol.toStringTag,{value:"Module"}))}(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))a(i);new MutationObserver(i=>{for(const s of i)if(s.type==="childList")for(const c of s.addedNodes)c.tagName==="LINK"&&c.rel==="modulepreload"&&a(c)}).observe(document,{childList:!0,subtree:!0});function n(i){const s={};return i.integrity&&(s.integrity=i.integrity),i.referrerPolicy&&(s.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?s.credentials="include":i.crossOrigin==="anonymous"?s.credentials="omit":s.credentials="same-origin",s}function a(i){if(i.ep)return;i.ep=!0;const s=n(i);fetch(i.href,s)}})();var Pu=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Tm(r){return r&&r.__esModule&&Object.prototype.hasOwnProperty.call(r,"default")?r.default:r}function Vv(r){if(Object.prototype.hasOwnProperty.call(r,"__esModule"))return r;var t=r.default;if(typeof t=="function"){var n=function a(){return this instanceof a?Reflect.construct(t,arguments,this.constructor):t.apply(this,arguments)};n.prototype=t.prototype}else n={};return Object.defineProperty(n,"__esModule",{value:!0}),Object.keys(r).forEach(function(a){var i=Object.getOwnPropertyDescriptor(r,a);Object.defineProperty(n,a,i.get?i:{enumerable:!0,get:function(){return r[a]}})}),n}var ku={exports:{}},Ma={},Du={exports:{}},Ge={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var hp;function Wv(){if(hp)return Ge;hp=1;var r=Symbol.for("react.element"),t=Symbol.for("react.portal"),n=Symbol.for("react.fragment"),a=Symbol.for("react.strict_mode"),i=Symbol.for("react.profiler"),s=Symbol.for("react.provider"),c=Symbol.for("react.context"),u=Symbol.for("react.forward_ref"),p=Symbol.for("react.suspense"),f=Symbol.for("react.memo"),h=Symbol.for("react.lazy"),x=Symbol.iterator;function y(_){return _===null||typeof _!="object"?null:(_=x&&_[x]||_["@@iterator"],typeof _=="function"?_:null)}var w={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},E=Object.assign,C={};function S(_,$,G){this.props=_,this.context=$,this.refs=C,this.updater=G||w}S.prototype.isReactComponent={},S.prototype.setState=function(_,$){if(typeof _!="object"&&typeof _!="function"&&_!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,_,$,"setState")},S.prototype.forceUpdate=function(_){this.updater.enqueueForceUpdate(this,_,"forceUpdate")};function P(){}P.prototype=S.prototype;function b(_,$,G){this.props=_,this.context=$,this.refs=C,this.updater=G||w}var A=b.prototype=new P;A.constructor=b,E(A,S.prototype),A.isPureReactComponent=!0;var T=Array.isArray,B=Object.prototype.hasOwnProperty,F={current:null},k={key:!0,ref:!0,__self:!0,__source:!0};function N(_,$,G){var K,ue={},me=null,Le=null;if($!=null)for(K in $.ref!==void 0&&(Le=$.ref),$.key!==void 0&&(me=""+$.key),$)B.call($,K)&&!k.hasOwnProperty(K)&&(ue[K]=$[K]);var he=arguments.length-2;if(he===1)ue.children=G;else if(1<he){for(var Ve=Array(he),ot=0;ot<he;ot++)Ve[ot]=arguments[ot+2];ue.children=Ve}if(_&&_.defaultProps)for(K in he=_.defaultProps,he)ue[K]===void 0&&(ue[K]=he[K]);return{$$typeof:r,type:_,key:me,ref:Le,props:ue,_owner:F.current}}function I(_,$){return{$$typeof:r,type:_.type,key:$,ref:_.ref,props:_.props,_owner:_._owner}}function L(_){return typeof _=="object"&&_!==null&&_.$$typeof===r}function M(_){var $={"=":"=0",":":"=2"};return"$"+_.replace(/[=:]/g,function(G){return $[G]})}var H=/\/+/g;function z(_,$){return typeof _=="object"&&_!==null&&_.key!=null?M(""+_.key):$.toString(36)}function Z(_,$,G,K,ue){var me=typeof _;(me==="undefined"||me==="boolean")&&(_=null);var Le=!1;if(_===null)Le=!0;else switch(me){case"string":case"number":Le=!0;break;case"object":switch(_.$$typeof){case r:case t:Le=!0}}if(Le)return Le=_,ue=ue(Le),_=K===""?"."+z(Le,0):K,T(ue)?(G="",_!=null&&(G=_.replace(H,"$&/")+"/"),Z(ue,$,G,"",function(ot){return ot})):ue!=null&&(L(ue)&&(ue=I(ue,G+(!ue.key||Le&&Le.key===ue.key?"":(""+ue.key).replace(H,"$&/")+"/")+_)),$.push(ue)),1;if(Le=0,K=K===""?".":K+":",T(_))for(var he=0;he<_.length;he++){me=_[he];var Ve=K+z(me,he);Le+=Z(me,$,G,Ve,ue)}else if(Ve=y(_),typeof Ve=="function")for(_=Ve.call(_),he=0;!(me=_.next()).done;)me=me.value,Ve=K+z(me,he++),Le+=Z(me,$,G,Ve,ue);else if(me==="object")throw $=String(_),Error("Objects are not valid as a React child (found: "+($==="[object Object]"?"object with keys {"+Object.keys(_).join(", ")+"}":$)+"). If you meant to render a collection of children, use an array instead.");return Le}function oe(_,$,G){if(_==null)return _;var K=[],ue=0;return Z(_,K,"","",function(me){return $.call(G,me,ue++)}),K}function re(_){if(_._status===-1){var $=_._result;$=$(),$.then(function(G){(_._status===0||_._status===-1)&&(_._status=1,_._result=G)},function(G){(_._status===0||_._status===-1)&&(_._status=2,_._result=G)}),_._status===-1&&(_._status=0,_._result=$)}if(_._status===1)return _._result.default;throw _._result}var se={current:null},Y={transition:null},te={ReactCurrentDispatcher:se,ReactCurrentBatchConfig:Y,ReactCurrentOwner:F};function ee(){throw Error("act(...) is not supported in production builds of React.")}return Ge.Children={map:oe,forEach:function(_,$,G){oe(_,function(){$.apply(this,arguments)},G)},count:function(_){var $=0;return oe(_,function(){$++}),$},toArray:function(_){return oe(_,function($){return $})||[]},only:function(_){if(!L(_))throw Error("React.Children.only expected to receive a single React element child.");return _}},Ge.Component=S,Ge.Fragment=n,Ge.Profiler=i,Ge.PureComponent=b,Ge.StrictMode=a,Ge.Suspense=p,Ge.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=te,Ge.act=ee,Ge.cloneElement=function(_,$,G){if(_==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+_+".");var K=E({},_.props),ue=_.key,me=_.ref,Le=_._owner;if($!=null){if($.ref!==void 0&&(me=$.ref,Le=F.current),$.key!==void 0&&(ue=""+$.key),_.type&&_.type.defaultProps)var he=_.type.defaultProps;for(Ve in $)B.call($,Ve)&&!k.hasOwnProperty(Ve)&&(K[Ve]=$[Ve]===void 0&&he!==void 0?he[Ve]:$[Ve])}var Ve=arguments.length-2;if(Ve===1)K.children=G;else if(1<Ve){he=Array(Ve);for(var ot=0;ot<Ve;ot++)he[ot]=arguments[ot+2];K.children=he}return{$$typeof:r,type:_.type,key:ue,ref:me,props:K,_owner:Le}},Ge.createContext=function(_){return _={$$typeof:c,_currentValue:_,_currentValue2:_,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},_.Provider={$$typeof:s,_context:_},_.Consumer=_},Ge.createElement=N,Ge.createFactory=function(_){var $=N.bind(null,_);return $.type=_,$},Ge.createRef=function(){return{current:null}},Ge.forwardRef=function(_){return{$$typeof:u,render:_}},Ge.isValidElement=L,Ge.lazy=function(_){return{$$typeof:h,_payload:{_status:-1,_result:_},_init:re}},Ge.memo=function(_,$){return{$$typeof:f,type:_,compare:$===void 0?null:$}},Ge.startTransition=function(_){var $=Y.transition;Y.transition={};try{_()}finally{Y.transition=$}},Ge.unstable_act=ee,Ge.useCallback=function(_,$){return se.current.useCallback(_,$)},Ge.useContext=function(_){return se.current.useContext(_)},Ge.useDebugValue=function(){},Ge.useDeferredValue=function(_){return se.current.useDeferredValue(_)},Ge.useEffect=function(_,$){return se.current.useEffect(_,$)},Ge.useId=function(){return se.current.useId()},Ge.useImperativeHandle=function(_,$,G){return se.current.useImperativeHandle(_,$,G)},Ge.useInsertionEffect=function(_,$){return se.current.useInsertionEffect(_,$)},Ge.useLayoutEffect=function(_,$){return se.current.useLayoutEffect(_,$)},Ge.useMemo=function(_,$){return se.current.useMemo(_,$)},Ge.useReducer=function(_,$,G){return se.current.useReducer(_,$,G)},Ge.useRef=function(_){return se.current.useRef(_)},Ge.useState=function(_){return se.current.useState(_)},Ge.useSyncExternalStore=function(_,$,G){return se.current.useSyncExternalStore(_,$,G)},Ge.useTransition=function(){return se.current.useTransition()},Ge.version="18.3.1",Ge}var mp;function Ll(){return mp||(mp=1,Du.exports=Wv()),Du.exports}/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var gp;function Gv(){if(gp)return Ma;gp=1;var r=Ll(),t=Symbol.for("react.element"),n=Symbol.for("react.fragment"),a=Object.prototype.hasOwnProperty,i=r.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,s={key:!0,ref:!0,__self:!0,__source:!0};function c(u,p,f){var h,x={},y=null,w=null;f!==void 0&&(y=""+f),p.key!==void 0&&(y=""+p.key),p.ref!==void 0&&(w=p.ref);for(h in p)a.call(p,h)&&!s.hasOwnProperty(h)&&(x[h]=p[h]);if(u&&u.defaultProps)for(h in p=u.defaultProps,p)x[h]===void 0&&(x[h]=p[h]);return{$$typeof:t,type:u,key:y,ref:w,props:x,_owner:i.current}}return Ma.Fragment=n,Ma.jsx=c,Ma.jsxs=c,Ma}var xp;function qv(){return xp||(xp=1,ku.exports=Gv()),ku.exports}var g=qv(),D=Ll();const Wa=Tm(D),Kv=Hv({__proto__:null,default:Wa},[D]);var ms={},Tu={exports:{}},sr={},_u={exports:{}},Lu={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var vp;function Xv(){return vp||(vp=1,(function(r){function t(Y,te){var ee=Y.length;Y.push(te);e:for(;0<ee;){var _=ee-1>>>1,$=Y[_];if(0<i($,te))Y[_]=te,Y[ee]=$,ee=_;else break e}}function n(Y){return Y.length===0?null:Y[0]}function a(Y){if(Y.length===0)return null;var te=Y[0],ee=Y.pop();if(ee!==te){Y[0]=ee;e:for(var _=0,$=Y.length,G=$>>>1;_<G;){var K=2*(_+1)-1,ue=Y[K],me=K+1,Le=Y[me];if(0>i(ue,ee))me<$&&0>i(Le,ue)?(Y[_]=Le,Y[me]=ee,_=me):(Y[_]=ue,Y[K]=ee,_=K);else if(me<$&&0>i(Le,ee))Y[_]=Le,Y[me]=ee,_=me;else break e}}return te}function i(Y,te){var ee=Y.sortIndex-te.sortIndex;return ee!==0?ee:Y.id-te.id}if(typeof performance=="object"&&typeof performance.now=="function"){var s=performance;r.unstable_now=function(){return s.now()}}else{var c=Date,u=c.now();r.unstable_now=function(){return c.now()-u}}var p=[],f=[],h=1,x=null,y=3,w=!1,E=!1,C=!1,S=typeof setTimeout=="function"?setTimeout:null,P=typeof clearTimeout=="function"?clearTimeout:null,b=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function A(Y){for(var te=n(f);te!==null;){if(te.callback===null)a(f);else if(te.startTime<=Y)a(f),te.sortIndex=te.expirationTime,t(p,te);else break;te=n(f)}}function T(Y){if(C=!1,A(Y),!E)if(n(p)!==null)E=!0,re(B);else{var te=n(f);te!==null&&se(T,te.startTime-Y)}}function B(Y,te){E=!1,C&&(C=!1,P(N),N=-1),w=!0;var ee=y;try{for(A(te),x=n(p);x!==null&&(!(x.expirationTime>te)||Y&&!M());){var _=x.callback;if(typeof _=="function"){x.callback=null,y=x.priorityLevel;var $=_(x.expirationTime<=te);te=r.unstable_now(),typeof $=="function"?x.callback=$:x===n(p)&&a(p),A(te)}else a(p);x=n(p)}if(x!==null)var G=!0;else{var K=n(f);K!==null&&se(T,K.startTime-te),G=!1}return G}finally{x=null,y=ee,w=!1}}var F=!1,k=null,N=-1,I=5,L=-1;function M(){return!(r.unstable_now()-L<I)}function H(){if(k!==null){var Y=r.unstable_now();L=Y;var te=!0;try{te=k(!0,Y)}finally{te?z():(F=!1,k=null)}}else F=!1}var z;if(typeof b=="function")z=function(){b(H)};else if(typeof MessageChannel<"u"){var Z=new MessageChannel,oe=Z.port2;Z.port1.onmessage=H,z=function(){oe.postMessage(null)}}else z=function(){S(H,0)};function re(Y){k=Y,F||(F=!0,z())}function se(Y,te){N=S(function(){Y(r.unstable_now())},te)}r.unstable_IdlePriority=5,r.unstable_ImmediatePriority=1,r.unstable_LowPriority=4,r.unstable_NormalPriority=3,r.unstable_Profiling=null,r.unstable_UserBlockingPriority=2,r.unstable_cancelCallback=function(Y){Y.callback=null},r.unstable_continueExecution=function(){E||w||(E=!0,re(B))},r.unstable_forceFrameRate=function(Y){0>Y||125<Y?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):I=0<Y?Math.floor(1e3/Y):5},r.unstable_getCurrentPriorityLevel=function(){return y},r.unstable_getFirstCallbackNode=function(){return n(p)},r.unstable_next=function(Y){switch(y){case 1:case 2:case 3:var te=3;break;default:te=y}var ee=y;y=te;try{return Y()}finally{y=ee}},r.unstable_pauseExecution=function(){},r.unstable_requestPaint=function(){},r.unstable_runWithPriority=function(Y,te){switch(Y){case 1:case 2:case 3:case 4:case 5:break;default:Y=3}var ee=y;y=Y;try{return te()}finally{y=ee}},r.unstable_scheduleCallback=function(Y,te,ee){var _=r.unstable_now();switch(typeof ee=="object"&&ee!==null?(ee=ee.delay,ee=typeof ee=="number"&&0<ee?_+ee:_):ee=_,Y){case 1:var $=-1;break;case 2:$=250;break;case 5:$=1073741823;break;case 4:$=1e4;break;default:$=5e3}return $=ee+$,Y={id:h++,callback:te,priorityLevel:Y,startTime:ee,expirationTime:$,sortIndex:-1},ee>_?(Y.sortIndex=ee,t(f,Y),n(p)===null&&Y===n(f)&&(C?(P(N),N=-1):C=!0,se(T,ee-_))):(Y.sortIndex=$,t(p,Y),E||w||(E=!0,re(B))),Y},r.unstable_shouldYield=M,r.unstable_wrapCallback=function(Y){var te=y;return function(){var ee=y;y=te;try{return Y.apply(this,arguments)}finally{y=ee}}}})(Lu)),Lu}var yp;function Yv(){return yp||(yp=1,_u.exports=Xv()),_u.exports}/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var wp;function Qv(){if(wp)return sr;wp=1;var r=Ll(),t=Yv();function n(e){for(var o="https://reactjs.org/docs/error-decoder.html?invariant="+e,l=1;l<arguments.length;l++)o+="&args[]="+encodeURIComponent(arguments[l]);return"Minified React error #"+e+"; visit "+o+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var a=new Set,i={};function s(e,o){c(e,o),c(e+"Capture",o)}function c(e,o){for(i[e]=o,e=0;e<o.length;e++)a.add(o[e])}var u=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),p=Object.prototype.hasOwnProperty,f=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,h={},x={};function y(e){return p.call(x,e)?!0:p.call(h,e)?!1:f.test(e)?x[e]=!0:(h[e]=!0,!1)}function w(e,o,l,d){if(l!==null&&l.type===0)return!1;switch(typeof o){case"function":case"symbol":return!0;case"boolean":return d?!1:l!==null?!l.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function E(e,o,l,d){if(o===null||typeof o>"u"||w(e,o,l,d))return!0;if(d)return!1;if(l!==null)switch(l.type){case 3:return!o;case 4:return o===!1;case 5:return isNaN(o);case 6:return isNaN(o)||1>o}return!1}function C(e,o,l,d,m,v,R){this.acceptsBooleans=o===2||o===3||o===4,this.attributeName=d,this.attributeNamespace=m,this.mustUseProperty=l,this.propertyName=e,this.type=o,this.sanitizeURL=v,this.removeEmptyString=R}var S={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){S[e]=new C(e,0,!1,e,null,!1,!1)}),[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var o=e[0];S[o]=new C(o,1,!1,e[1],null,!1,!1)}),["contentEditable","draggable","spellCheck","value"].forEach(function(e){S[e]=new C(e,2,!1,e.toLowerCase(),null,!1,!1)}),["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){S[e]=new C(e,2,!1,e,null,!1,!1)}),"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){S[e]=new C(e,3,!1,e.toLowerCase(),null,!1,!1)}),["checked","multiple","muted","selected"].forEach(function(e){S[e]=new C(e,3,!0,e,null,!1,!1)}),["capture","download"].forEach(function(e){S[e]=new C(e,4,!1,e,null,!1,!1)}),["cols","rows","size","span"].forEach(function(e){S[e]=new C(e,6,!1,e,null,!1,!1)}),["rowSpan","start"].forEach(function(e){S[e]=new C(e,5,!1,e.toLowerCase(),null,!1,!1)});var P=/[\-:]([a-z])/g;function b(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var o=e.replace(P,b);S[o]=new C(o,1,!1,e,null,!1,!1)}),"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var o=e.replace(P,b);S[o]=new C(o,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)}),["xml:base","xml:lang","xml:space"].forEach(function(e){var o=e.replace(P,b);S[o]=new C(o,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)}),["tabIndex","crossOrigin"].forEach(function(e){S[e]=new C(e,1,!1,e.toLowerCase(),null,!1,!1)}),S.xlinkHref=new C("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1),["src","href","action","formAction"].forEach(function(e){S[e]=new C(e,1,!1,e.toLowerCase(),null,!0,!0)});function A(e,o,l,d){var m=S.hasOwnProperty(o)?S[o]:null;(m!==null?m.type!==0:d||!(2<o.length)||o[0]!=="o"&&o[0]!=="O"||o[1]!=="n"&&o[1]!=="N")&&(E(o,l,m,d)&&(l=null),d||m===null?y(o)&&(l===null?e.removeAttribute(o):e.setAttribute(o,""+l)):m.mustUseProperty?e[m.propertyName]=l===null?m.type===3?!1:"":l:(o=m.attributeName,d=m.attributeNamespace,l===null?e.removeAttribute(o):(m=m.type,l=m===3||m===4&&l===!0?"":""+l,d?e.setAttributeNS(d,o,l):e.setAttribute(o,l))))}var T=r.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,B=Symbol.for("react.element"),F=Symbol.for("react.portal"),k=Symbol.for("react.fragment"),N=Symbol.for("react.strict_mode"),I=Symbol.for("react.profiler"),L=Symbol.for("react.provider"),M=Symbol.for("react.context"),H=Symbol.for("react.forward_ref"),z=Symbol.for("react.suspense"),Z=Symbol.for("react.suspense_list"),oe=Symbol.for("react.memo"),re=Symbol.for("react.lazy"),se=Symbol.for("react.offscreen"),Y=Symbol.iterator;function te(e){return e===null||typeof e!="object"?null:(e=Y&&e[Y]||e["@@iterator"],typeof e=="function"?e:null)}var ee=Object.assign,_;function $(e){if(_===void 0)try{throw Error()}catch(l){var o=l.stack.trim().match(/\n( *(at )?)/);_=o&&o[1]||""}return`
`+_+e}var G=!1;function K(e,o){if(!e||G)return"";G=!0;var l=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(o)if(o=function(){throw Error()},Object.defineProperty(o.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(o,[])}catch(J){var d=J}Reflect.construct(e,[],o)}else{try{o.call()}catch(J){d=J}e.call(o.prototype)}else{try{throw Error()}catch(J){d=J}e()}}catch(J){if(J&&d&&typeof J.stack=="string"){for(var m=J.stack.split(`
`),v=d.stack.split(`
`),R=m.length-1,j=v.length-1;1<=R&&0<=j&&m[R]!==v[j];)j--;for(;1<=R&&0<=j;R--,j--)if(m[R]!==v[j]){if(R!==1||j!==1)do if(R--,j--,0>j||m[R]!==v[j]){var U=`
`+m[R].replace(" at new "," at ");return e.displayName&&U.includes("<anonymous>")&&(U=U.replace("<anonymous>",e.displayName)),U}while(1<=R&&0<=j);break}}}finally{G=!1,Error.prepareStackTrace=l}return(e=e?e.displayName||e.name:"")?$(e):""}function ue(e){switch(e.tag){case 5:return $(e.type);case 16:return $("Lazy");case 13:return $("Suspense");case 19:return $("SuspenseList");case 0:case 2:case 15:return e=K(e.type,!1),e;case 11:return e=K(e.type.render,!1),e;case 1:return e=K(e.type,!0),e;default:return""}}function me(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case k:return"Fragment";case F:return"Portal";case I:return"Profiler";case N:return"StrictMode";case z:return"Suspense";case Z:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case M:return(e.displayName||"Context")+".Consumer";case L:return(e._context.displayName||"Context")+".Provider";case H:var o=e.render;return e=e.displayName,e||(e=o.displayName||o.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case oe:return o=e.displayName||null,o!==null?o:me(e.type)||"Memo";case re:o=e._payload,e=e._init;try{return me(e(o))}catch{}}return null}function Le(e){var o=e.type;switch(e.tag){case 24:return"Cache";case 9:return(o.displayName||"Context")+".Consumer";case 10:return(o._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=o.render,e=e.displayName||e.name||"",o.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return o;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return me(o);case 8:return o===N?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof o=="function")return o.displayName||o.name||null;if(typeof o=="string")return o}return null}function he(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function Ve(e){var o=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(o==="checkbox"||o==="radio")}function ot(e){var o=Ve(e)?"checked":"value",l=Object.getOwnPropertyDescriptor(e.constructor.prototype,o),d=""+e[o];if(!e.hasOwnProperty(o)&&typeof l<"u"&&typeof l.get=="function"&&typeof l.set=="function"){var m=l.get,v=l.set;return Object.defineProperty(e,o,{configurable:!0,get:function(){return m.call(this)},set:function(R){d=""+R,v.call(this,R)}}),Object.defineProperty(e,o,{enumerable:l.enumerable}),{getValue:function(){return d},setValue:function(R){d=""+R},stopTracking:function(){e._valueTracker=null,delete e[o]}}}}function Vt(e){e._valueTracker||(e._valueTracker=ot(e))}function We(e){if(!e)return!1;var o=e._valueTracker;if(!o)return!0;var l=o.getValue(),d="";return e&&(d=Ve(e)?e.checked?"true":"false":e.value),e=d,e!==l?(o.setValue(e),!0):!1}function Ee(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function ge(e,o){var l=o.checked;return ee({},o,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:l??e._wrapperState.initialChecked})}function je(e,o){var l=o.defaultValue==null?"":o.defaultValue,d=o.checked!=null?o.checked:o.defaultChecked;l=he(o.value!=null?o.value:l),e._wrapperState={initialChecked:d,initialValue:l,controlled:o.type==="checkbox"||o.type==="radio"?o.checked!=null:o.value!=null}}function Ye(e,o){o=o.checked,o!=null&&A(e,"checked",o,!1)}function tt(e,o){Ye(e,o);var l=he(o.value),d=o.type;if(l!=null)d==="number"?(l===0&&e.value===""||e.value!=l)&&(e.value=""+l):e.value!==""+l&&(e.value=""+l);else if(d==="submit"||d==="reset"){e.removeAttribute("value");return}o.hasOwnProperty("value")?ke(e,o.type,l):o.hasOwnProperty("defaultValue")&&ke(e,o.type,he(o.defaultValue)),o.checked==null&&o.defaultChecked!=null&&(e.defaultChecked=!!o.defaultChecked)}function Dt(e,o,l){if(o.hasOwnProperty("value")||o.hasOwnProperty("defaultValue")){var d=o.type;if(!(d!=="submit"&&d!=="reset"||o.value!==void 0&&o.value!==null))return;o=""+e._wrapperState.initialValue,l||o===e.value||(e.value=o),e.defaultValue=o}l=e.name,l!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,l!==""&&(e.name=l)}function ke(e,o,l){(o!=="number"||Ee(e.ownerDocument)!==e)&&(l==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+l&&(e.defaultValue=""+l))}var xe=Array.isArray;function ye(e,o,l,d){if(e=e.options,o){o={};for(var m=0;m<l.length;m++)o["$"+l[m]]=!0;for(l=0;l<e.length;l++)m=o.hasOwnProperty("$"+e[l].value),e[l].selected!==m&&(e[l].selected=m),m&&d&&(e[l].defaultSelected=!0)}else{for(l=""+he(l),o=null,m=0;m<e.length;m++){if(e[m].value===l){e[m].selected=!0,d&&(e[m].defaultSelected=!0);return}o!==null||e[m].disabled||(o=e[m])}o!==null&&(o.selected=!0)}}function Fe(e,o){if(o.dangerouslySetInnerHTML!=null)throw Error(n(91));return ee({},o,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function He(e,o){var l=o.value;if(l==null){if(l=o.children,o=o.defaultValue,l!=null){if(o!=null)throw Error(n(92));if(xe(l)){if(1<l.length)throw Error(n(93));l=l[0]}o=l}o==null&&(o=""),l=o}e._wrapperState={initialValue:he(l)}}function Oe(e,o){var l=he(o.value),d=he(o.defaultValue);l!=null&&(l=""+l,l!==e.value&&(e.value=l),o.defaultValue==null&&e.defaultValue!==l&&(e.defaultValue=l)),d!=null&&(e.defaultValue=""+d)}function ct(e){var o=e.textContent;o===e._wrapperState.initialValue&&o!==""&&o!==null&&(e.value=o)}function rt(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Rt(e,o){return e==null||e==="http://www.w3.org/1999/xhtml"?rt(o):e==="http://www.w3.org/2000/svg"&&o==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var Ke,mt=(function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(o,l,d,m){MSApp.execUnsafeLocalFunction(function(){return e(o,l,d,m)})}:e})(function(e,o){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=o;else{for(Ke=Ke||document.createElement("div"),Ke.innerHTML="<svg>"+o.valueOf().toString()+"</svg>",o=Ke.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;o.firstChild;)e.appendChild(o.firstChild)}});function Tt(e,o){if(o){var l=e.firstChild;if(l&&l===e.lastChild&&l.nodeType===3){l.nodeValue=o;return}}e.textContent=o}var Ft={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},cr=["Webkit","ms","Moz","O"];Object.keys(Ft).forEach(function(e){cr.forEach(function(o){o=o+e.charAt(0).toUpperCase()+e.substring(1),Ft[o]=Ft[e]})});function Po(e,o,l){return o==null||typeof o=="boolean"||o===""?"":l||typeof o!="number"||o===0||Ft.hasOwnProperty(e)&&Ft[e]?(""+o).trim():o+"px"}function Yt(e,o){e=e.style;for(var l in o)if(o.hasOwnProperty(l)){var d=l.indexOf("--")===0,m=Po(l,o[l],d);l==="float"&&(l="cssFloat"),d?e.setProperty(l,m):e[l]=m}}var Zn=ee({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function mn(e,o){if(o){if(Zn[e]&&(o.children!=null||o.dangerouslySetInnerHTML!=null))throw Error(n(137,e));if(o.dangerouslySetInnerHTML!=null){if(o.children!=null)throw Error(n(60));if(typeof o.dangerouslySetInnerHTML!="object"||!("__html"in o.dangerouslySetInnerHTML))throw Error(n(61))}if(o.style!=null&&typeof o.style!="object")throw Error(n(62))}}function gn(e,o){if(e.indexOf("-")===-1)return typeof o.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var xn=null;function vn(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Pr=null,kr=null,ur=null;function ko(e){if(e=Sa(e)){if(typeof Pr!="function")throw Error(n(280));var o=e.stateNode;o&&(o=Ti(o),Pr(e.stateNode,e.type,o))}}function eo(e){kr?ur?ur.push(e):ur=[e]:kr=e}function Do(){if(kr){var e=kr,o=ur;if(ur=kr=null,ko(e),o)for(e=0;e<o.length;e++)ko(o[e])}}function yn(e,o){return e(o)}function Kr(){}var Or=!1;function wn(e,o,l){if(Or)return e(o,l);Or=!0;try{return yn(e,o,l)}finally{Or=!1,(kr!==null||ur!==null)&&(Kr(),Do())}}function Xr(e,o){var l=e.stateNode;if(l===null)return null;var d=Ti(l);if(d===null)return null;l=d[o];e:switch(o){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(d=!d.disabled)||(e=e.type,d=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!d;break e;default:e=!1}if(e)return null;if(l&&typeof l!="function")throw Error(n(231,o,typeof l));return l}var En=!1;if(u)try{var O={};Object.defineProperty(O,"passive",{get:function(){En=!0}}),window.addEventListener("test",O,O),window.removeEventListener("test",O,O)}catch{En=!1}function W(e,o,l,d,m,v,R,j,U){var J=Array.prototype.slice.call(arguments,3);try{o.apply(l,J)}catch(le){this.onError(le)}}var Q=!1,ae=null,fe=!1,De=null,Ne={onError:function(e){Q=!0,ae=e}};function Ce(e,o,l,d,m,v,R,j,U){Q=!1,ae=null,W.apply(Ne,arguments)}function Se(e,o,l,d,m,v,R,j,U){if(Ce.apply(this,arguments),Q){if(Q){var J=ae;Q=!1,ae=null}else throw Error(n(198));fe||(fe=!0,De=J)}}function Ae(e){var o=e,l=e;if(e.alternate)for(;o.return;)o=o.return;else{e=o;do o=e,(o.flags&4098)!==0&&(l=o.return),e=o.return;while(e)}return o.tag===3?l:null}function Ue(e){if(e.tag===13){var o=e.memoizedState;if(o===null&&(e=e.alternate,e!==null&&(o=e.memoizedState)),o!==null)return o.dehydrated}return null}function _e(e){if(Ae(e)!==e)throw Error(n(188))}function ze(e){var o=e.alternate;if(!o){if(o=Ae(e),o===null)throw Error(n(188));return o!==e?null:e}for(var l=e,d=o;;){var m=l.return;if(m===null)break;var v=m.alternate;if(v===null){if(d=m.return,d!==null){l=d;continue}break}if(m.child===v.child){for(v=m.child;v;){if(v===l)return _e(m),e;if(v===d)return _e(m),o;v=v.sibling}throw Error(n(188))}if(l.return!==d.return)l=m,d=v;else{for(var R=!1,j=m.child;j;){if(j===l){R=!0,l=m,d=v;break}if(j===d){R=!0,d=m,l=v;break}j=j.sibling}if(!R){for(j=v.child;j;){if(j===l){R=!0,l=v,d=m;break}if(j===d){R=!0,d=v,l=m;break}j=j.sibling}if(!R)throw Error(n(189))}}if(l.alternate!==d)throw Error(n(190))}if(l.tag!==3)throw Error(n(188));return l.stateNode.current===l?e:o}function Xe(e){return e=ze(e),e!==null?_t(e):null}function _t(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var o=_t(e);if(o!==null)return o;e=e.sibling}return null}var At=t.unstable_scheduleCallback,Bt=t.unstable_cancelCallback,nt=t.unstable_shouldYield,tr=t.unstable_requestPaint,at=t.unstable_now,to=t.unstable_getCurrentPriorityLevel,xr=t.unstable_ImmediatePriority,dr=t.unstable_UserBlockingPriority,Cn=t.unstable_NormalPriority,ro=t.unstable_LowPriority,Mr=t.unstable_IdlePriority,Yr=null,Qt=null;function Qe(e){if(Qt&&typeof Qt.onCommitFiberRoot=="function")try{Qt.onCommitFiberRoot(Yr,e,void 0,(e.current.flags&128)===128)}catch{}}var ut=Math.clz32?Math.clz32:vt,bn=Math.log,Qr=Math.LN2;function vt(e){return e>>>=0,e===0?32:31-(bn(e)/Qr|0)|0}var Jr=64,no=4194304;function oo(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function hi(e,o){var l=e.pendingLanes;if(l===0)return 0;var d=0,m=e.suspendedLanes,v=e.pingedLanes,R=l&268435455;if(R!==0){var j=R&~m;j!==0?d=oo(j):(v&=R,v!==0&&(d=oo(v)))}else R=l&~m,R!==0?d=oo(R):v!==0&&(d=oo(v));if(d===0)return 0;if(o!==0&&o!==d&&(o&m)===0&&(m=d&-d,v=o&-o,m>=v||m===16&&(v&4194240)!==0))return o;if((d&4)!==0&&(d|=l&16),o=e.entangledLanes,o!==0)for(e=e.entanglements,o&=d;0<o;)l=31-ut(o),m=1<<l,d|=e[l],o&=~m;return d}function ix(e,o){switch(e){case 1:case 2:case 4:return o+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return o+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function sx(e,o){for(var l=e.suspendedLanes,d=e.pingedLanes,m=e.expirationTimes,v=e.pendingLanes;0<v;){var R=31-ut(v),j=1<<R,U=m[R];U===-1?((j&l)===0||(j&d)!==0)&&(m[R]=ix(j,o)):U<=o&&(e.expiredLanes|=j),v&=~j}}function Kl(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Vd(){var e=Jr;return Jr<<=1,(Jr&4194240)===0&&(Jr=64),e}function Xl(e){for(var o=[],l=0;31>l;l++)o.push(e);return o}function sa(e,o,l){e.pendingLanes|=o,o!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,o=31-ut(o),e[o]=l}function lx(e,o){var l=e.pendingLanes&~o;e.pendingLanes=o,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=o,e.mutableReadLanes&=o,e.entangledLanes&=o,o=e.entanglements;var d=e.eventTimes;for(e=e.expirationTimes;0<l;){var m=31-ut(l),v=1<<m;o[m]=0,d[m]=-1,e[m]=-1,l&=~v}}function Yl(e,o){var l=e.entangledLanes|=o;for(e=e.entanglements;l;){var d=31-ut(l),m=1<<d;m&o|e[d]&o&&(e[d]|=o),l&=~m}}var it=0;function Wd(e){return e&=-e,1<e?4<e?(e&268435455)!==0?16:536870912:4:1}var Gd,Ql,qd,Kd,Xd,Jl=!1,mi=[],Sn=null,Rn=null,An=null,la=new Map,ca=new Map,Pn=[],cx="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Yd(e,o){switch(e){case"focusin":case"focusout":Sn=null;break;case"dragenter":case"dragleave":Rn=null;break;case"mouseover":case"mouseout":An=null;break;case"pointerover":case"pointerout":la.delete(o.pointerId);break;case"gotpointercapture":case"lostpointercapture":ca.delete(o.pointerId)}}function ua(e,o,l,d,m,v){return e===null||e.nativeEvent!==v?(e={blockedOn:o,domEventName:l,eventSystemFlags:d,nativeEvent:v,targetContainers:[m]},o!==null&&(o=Sa(o),o!==null&&Ql(o)),e):(e.eventSystemFlags|=d,o=e.targetContainers,m!==null&&o.indexOf(m)===-1&&o.push(m),e)}function ux(e,o,l,d,m){switch(o){case"focusin":return Sn=ua(Sn,e,o,l,d,m),!0;case"dragenter":return Rn=ua(Rn,e,o,l,d,m),!0;case"mouseover":return An=ua(An,e,o,l,d,m),!0;case"pointerover":var v=m.pointerId;return la.set(v,ua(la.get(v)||null,e,o,l,d,m)),!0;case"gotpointercapture":return v=m.pointerId,ca.set(v,ua(ca.get(v)||null,e,o,l,d,m)),!0}return!1}function Qd(e){var o=ao(e.target);if(o!==null){var l=Ae(o);if(l!==null){if(o=l.tag,o===13){if(o=Ue(l),o!==null){e.blockedOn=o,Xd(e.priority,function(){qd(l)});return}}else if(o===3&&l.stateNode.current.memoizedState.isDehydrated){e.blockedOn=l.tag===3?l.stateNode.containerInfo:null;return}}}e.blockedOn=null}function gi(e){if(e.blockedOn!==null)return!1;for(var o=e.targetContainers;0<o.length;){var l=ec(e.domEventName,e.eventSystemFlags,o[0],e.nativeEvent);if(l===null){l=e.nativeEvent;var d=new l.constructor(l.type,l);xn=d,l.target.dispatchEvent(d),xn=null}else return o=Sa(l),o!==null&&Ql(o),e.blockedOn=l,!1;o.shift()}return!0}function Jd(e,o,l){gi(e)&&l.delete(o)}function dx(){Jl=!1,Sn!==null&&gi(Sn)&&(Sn=null),Rn!==null&&gi(Rn)&&(Rn=null),An!==null&&gi(An)&&(An=null),la.forEach(Jd),ca.forEach(Jd)}function da(e,o){e.blockedOn===o&&(e.blockedOn=null,Jl||(Jl=!0,t.unstable_scheduleCallback(t.unstable_NormalPriority,dx)))}function fa(e){function o(m){return da(m,e)}if(0<mi.length){da(mi[0],e);for(var l=1;l<mi.length;l++){var d=mi[l];d.blockedOn===e&&(d.blockedOn=null)}}for(Sn!==null&&da(Sn,e),Rn!==null&&da(Rn,e),An!==null&&da(An,e),la.forEach(o),ca.forEach(o),l=0;l<Pn.length;l++)d=Pn[l],d.blockedOn===e&&(d.blockedOn=null);for(;0<Pn.length&&(l=Pn[0],l.blockedOn===null);)Qd(l),l.blockedOn===null&&Pn.shift()}var To=T.ReactCurrentBatchConfig,xi=!0;function fx(e,o,l,d){var m=it,v=To.transition;To.transition=null;try{it=1,Zl(e,o,l,d)}finally{it=m,To.transition=v}}function px(e,o,l,d){var m=it,v=To.transition;To.transition=null;try{it=4,Zl(e,o,l,d)}finally{it=m,To.transition=v}}function Zl(e,o,l,d){if(xi){var m=ec(e,o,l,d);if(m===null)xc(e,o,d,vi,l),Yd(e,d);else if(ux(m,e,o,l,d))d.stopPropagation();else if(Yd(e,d),o&4&&-1<cx.indexOf(e)){for(;m!==null;){var v=Sa(m);if(v!==null&&Gd(v),v=ec(e,o,l,d),v===null&&xc(e,o,d,vi,l),v===m)break;m=v}m!==null&&d.stopPropagation()}else xc(e,o,d,null,l)}}var vi=null;function ec(e,o,l,d){if(vi=null,e=vn(d),e=ao(e),e!==null)if(o=Ae(e),o===null)e=null;else if(l=o.tag,l===13){if(e=Ue(o),e!==null)return e;e=null}else if(l===3){if(o.stateNode.current.memoizedState.isDehydrated)return o.tag===3?o.stateNode.containerInfo:null;e=null}else o!==e&&(e=null);return vi=e,null}function Zd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(to()){case xr:return 1;case dr:return 4;case Cn:case ro:return 16;case Mr:return 536870912;default:return 16}default:return 16}}var kn=null,tc=null,yi=null;function ef(){if(yi)return yi;var e,o=tc,l=o.length,d,m="value"in kn?kn.value:kn.textContent,v=m.length;for(e=0;e<l&&o[e]===m[e];e++);var R=l-e;for(d=1;d<=R&&o[l-d]===m[v-d];d++);return yi=m.slice(e,1<d?1-d:void 0)}function wi(e){var o=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&o===13&&(e=13)):e=o,e===10&&(e=13),32<=e||e===13?e:0}function Ei(){return!0}function tf(){return!1}function fr(e){function o(l,d,m,v,R){this._reactName=l,this._targetInst=m,this.type=d,this.nativeEvent=v,this.target=R,this.currentTarget=null;for(var j in e)e.hasOwnProperty(j)&&(l=e[j],this[j]=l?l(v):v[j]);return this.isDefaultPrevented=(v.defaultPrevented!=null?v.defaultPrevented:v.returnValue===!1)?Ei:tf,this.isPropagationStopped=tf,this}return ee(o.prototype,{preventDefault:function(){this.defaultPrevented=!0;var l=this.nativeEvent;l&&(l.preventDefault?l.preventDefault():typeof l.returnValue!="unknown"&&(l.returnValue=!1),this.isDefaultPrevented=Ei)},stopPropagation:function(){var l=this.nativeEvent;l&&(l.stopPropagation?l.stopPropagation():typeof l.cancelBubble!="unknown"&&(l.cancelBubble=!0),this.isPropagationStopped=Ei)},persist:function(){},isPersistent:Ei}),o}var _o={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},rc=fr(_o),pa=ee({},_o,{view:0,detail:0}),hx=fr(pa),nc,oc,ha,Ci=ee({},pa,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:ic,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==ha&&(ha&&e.type==="mousemove"?(nc=e.screenX-ha.screenX,oc=e.screenY-ha.screenY):oc=nc=0,ha=e),nc)},movementY:function(e){return"movementY"in e?e.movementY:oc}}),rf=fr(Ci),mx=ee({},Ci,{dataTransfer:0}),gx=fr(mx),xx=ee({},pa,{relatedTarget:0}),ac=fr(xx),vx=ee({},_o,{animationName:0,elapsedTime:0,pseudoElement:0}),yx=fr(vx),wx=ee({},_o,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),Ex=fr(wx),Cx=ee({},_o,{data:0}),nf=fr(Cx),bx={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Sx={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Rx={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function Ax(e){var o=this.nativeEvent;return o.getModifierState?o.getModifierState(e):(e=Rx[e])?!!o[e]:!1}function ic(){return Ax}var Px=ee({},pa,{key:function(e){if(e.key){var o=bx[e.key]||e.key;if(o!=="Unidentified")return o}return e.type==="keypress"?(e=wi(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Sx[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:ic,charCode:function(e){return e.type==="keypress"?wi(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?wi(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),kx=fr(Px),Dx=ee({},Ci,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),of=fr(Dx),Tx=ee({},pa,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:ic}),_x=fr(Tx),Lx=ee({},_o,{propertyName:0,elapsedTime:0,pseudoElement:0}),Fx=fr(Lx),Nx=ee({},Ci,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),Ix=fr(Nx),Bx=[9,13,27,32],sc=u&&"CompositionEvent"in window,ma=null;u&&"documentMode"in document&&(ma=document.documentMode);var jx=u&&"TextEvent"in window&&!ma,af=u&&(!sc||ma&&8<ma&&11>=ma),sf=" ",lf=!1;function cf(e,o){switch(e){case"keyup":return Bx.indexOf(o.keyCode)!==-1;case"keydown":return o.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function uf(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Lo=!1;function Ox(e,o){switch(e){case"compositionend":return uf(o);case"keypress":return o.which!==32?null:(lf=!0,sf);case"textInput":return e=o.data,e===sf&&lf?null:e;default:return null}}function Mx(e,o){if(Lo)return e==="compositionend"||!sc&&cf(e,o)?(e=ef(),yi=tc=kn=null,Lo=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(o.ctrlKey||o.altKey||o.metaKey)||o.ctrlKey&&o.altKey){if(o.char&&1<o.char.length)return o.char;if(o.which)return String.fromCharCode(o.which)}return null;case"compositionend":return af&&o.locale!=="ko"?null:o.data;default:return null}}var $x={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function df(e){var o=e&&e.nodeName&&e.nodeName.toLowerCase();return o==="input"?!!$x[e.type]:o==="textarea"}function ff(e,o,l,d){eo(d),o=Pi(o,"onChange"),0<o.length&&(l=new rc("onChange","change",null,l,d),e.push({event:l,listeners:o}))}var ga=null,xa=null;function Ux(e){Tf(e,0)}function bi(e){var o=jo(e);if(We(o))return e}function zx(e,o){if(e==="change")return o}var pf=!1;if(u){var lc;if(u){var cc="oninput"in document;if(!cc){var hf=document.createElement("div");hf.setAttribute("oninput","return;"),cc=typeof hf.oninput=="function"}lc=cc}else lc=!1;pf=lc&&(!document.documentMode||9<document.documentMode)}function mf(){ga&&(ga.detachEvent("onpropertychange",gf),xa=ga=null)}function gf(e){if(e.propertyName==="value"&&bi(xa)){var o=[];ff(o,xa,e,vn(e)),wn(Ux,o)}}function Hx(e,o,l){e==="focusin"?(mf(),ga=o,xa=l,ga.attachEvent("onpropertychange",gf)):e==="focusout"&&mf()}function Vx(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return bi(xa)}function Wx(e,o){if(e==="click")return bi(o)}function Gx(e,o){if(e==="input"||e==="change")return bi(o)}function qx(e,o){return e===o&&(e!==0||1/e===1/o)||e!==e&&o!==o}var Dr=typeof Object.is=="function"?Object.is:qx;function va(e,o){if(Dr(e,o))return!0;if(typeof e!="object"||e===null||typeof o!="object"||o===null)return!1;var l=Object.keys(e),d=Object.keys(o);if(l.length!==d.length)return!1;for(d=0;d<l.length;d++){var m=l[d];if(!p.call(o,m)||!Dr(e[m],o[m]))return!1}return!0}function xf(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function vf(e,o){var l=xf(e);e=0;for(var d;l;){if(l.nodeType===3){if(d=e+l.textContent.length,e<=o&&d>=o)return{node:l,offset:o-e};e=d}e:{for(;l;){if(l.nextSibling){l=l.nextSibling;break e}l=l.parentNode}l=void 0}l=xf(l)}}function yf(e,o){return e&&o?e===o?!0:e&&e.nodeType===3?!1:o&&o.nodeType===3?yf(e,o.parentNode):"contains"in e?e.contains(o):e.compareDocumentPosition?!!(e.compareDocumentPosition(o)&16):!1:!1}function wf(){for(var e=window,o=Ee();o instanceof e.HTMLIFrameElement;){try{var l=typeof o.contentWindow.location.href=="string"}catch{l=!1}if(l)e=o.contentWindow;else break;o=Ee(e.document)}return o}function uc(e){var o=e&&e.nodeName&&e.nodeName.toLowerCase();return o&&(o==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||o==="textarea"||e.contentEditable==="true")}function Kx(e){var o=wf(),l=e.focusedElem,d=e.selectionRange;if(o!==l&&l&&l.ownerDocument&&yf(l.ownerDocument.documentElement,l)){if(d!==null&&uc(l)){if(o=d.start,e=d.end,e===void 0&&(e=o),"selectionStart"in l)l.selectionStart=o,l.selectionEnd=Math.min(e,l.value.length);else if(e=(o=l.ownerDocument||document)&&o.defaultView||window,e.getSelection){e=e.getSelection();var m=l.textContent.length,v=Math.min(d.start,m);d=d.end===void 0?v:Math.min(d.end,m),!e.extend&&v>d&&(m=d,d=v,v=m),m=vf(l,v);var R=vf(l,d);m&&R&&(e.rangeCount!==1||e.anchorNode!==m.node||e.anchorOffset!==m.offset||e.focusNode!==R.node||e.focusOffset!==R.offset)&&(o=o.createRange(),o.setStart(m.node,m.offset),e.removeAllRanges(),v>d?(e.addRange(o),e.extend(R.node,R.offset)):(o.setEnd(R.node,R.offset),e.addRange(o)))}}for(o=[],e=l;e=e.parentNode;)e.nodeType===1&&o.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof l.focus=="function"&&l.focus(),l=0;l<o.length;l++)e=o[l],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Xx=u&&"documentMode"in document&&11>=document.documentMode,Fo=null,dc=null,ya=null,fc=!1;function Ef(e,o,l){var d=l.window===l?l.document:l.nodeType===9?l:l.ownerDocument;fc||Fo==null||Fo!==Ee(d)||(d=Fo,"selectionStart"in d&&uc(d)?d={start:d.selectionStart,end:d.selectionEnd}:(d=(d.ownerDocument&&d.ownerDocument.defaultView||window).getSelection(),d={anchorNode:d.anchorNode,anchorOffset:d.anchorOffset,focusNode:d.focusNode,focusOffset:d.focusOffset}),ya&&va(ya,d)||(ya=d,d=Pi(dc,"onSelect"),0<d.length&&(o=new rc("onSelect","select",null,o,l),e.push({event:o,listeners:d}),o.target=Fo)))}function Si(e,o){var l={};return l[e.toLowerCase()]=o.toLowerCase(),l["Webkit"+e]="webkit"+o,l["Moz"+e]="moz"+o,l}var No={animationend:Si("Animation","AnimationEnd"),animationiteration:Si("Animation","AnimationIteration"),animationstart:Si("Animation","AnimationStart"),transitionend:Si("Transition","TransitionEnd")},pc={},Cf={};u&&(Cf=document.createElement("div").style,"AnimationEvent"in window||(delete No.animationend.animation,delete No.animationiteration.animation,delete No.animationstart.animation),"TransitionEvent"in window||delete No.transitionend.transition);function Ri(e){if(pc[e])return pc[e];if(!No[e])return e;var o=No[e],l;for(l in o)if(o.hasOwnProperty(l)&&l in Cf)return pc[e]=o[l];return e}var bf=Ri("animationend"),Sf=Ri("animationiteration"),Rf=Ri("animationstart"),Af=Ri("transitionend"),Pf=new Map,kf="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function Dn(e,o){Pf.set(e,o),s(o,[e])}for(var hc=0;hc<kf.length;hc++){var mc=kf[hc],Yx=mc.toLowerCase(),Qx=mc[0].toUpperCase()+mc.slice(1);Dn(Yx,"on"+Qx)}Dn(bf,"onAnimationEnd"),Dn(Sf,"onAnimationIteration"),Dn(Rf,"onAnimationStart"),Dn("dblclick","onDoubleClick"),Dn("focusin","onFocus"),Dn("focusout","onBlur"),Dn(Af,"onTransitionEnd"),c("onMouseEnter",["mouseout","mouseover"]),c("onMouseLeave",["mouseout","mouseover"]),c("onPointerEnter",["pointerout","pointerover"]),c("onPointerLeave",["pointerout","pointerover"]),s("onChange","change click focusin focusout input keydown keyup selectionchange".split(" ")),s("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" ")),s("onBeforeInput",["compositionend","keypress","textInput","paste"]),s("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" ")),s("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" ")),s("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var wa="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Jx=new Set("cancel close invalid load scroll toggle".split(" ").concat(wa));function Df(e,o,l){var d=e.type||"unknown-event";e.currentTarget=l,Se(d,o,void 0,e),e.currentTarget=null}function Tf(e,o){o=(o&4)!==0;for(var l=0;l<e.length;l++){var d=e[l],m=d.event;d=d.listeners;e:{var v=void 0;if(o)for(var R=d.length-1;0<=R;R--){var j=d[R],U=j.instance,J=j.currentTarget;if(j=j.listener,U!==v&&m.isPropagationStopped())break e;Df(m,j,J),v=U}else for(R=0;R<d.length;R++){if(j=d[R],U=j.instance,J=j.currentTarget,j=j.listener,U!==v&&m.isPropagationStopped())break e;Df(m,j,J),v=U}}}if(fe)throw e=De,fe=!1,De=null,e}function gt(e,o){var l=o[bc];l===void 0&&(l=o[bc]=new Set);var d=e+"__bubble";l.has(d)||(_f(o,e,2,!1),l.add(d))}function gc(e,o,l){var d=0;o&&(d|=4),_f(l,e,d,o)}var Ai="_reactListening"+Math.random().toString(36).slice(2);function Ea(e){if(!e[Ai]){e[Ai]=!0,a.forEach(function(l){l!=="selectionchange"&&(Jx.has(l)||gc(l,!1,e),gc(l,!0,e))});var o=e.nodeType===9?e:e.ownerDocument;o===null||o[Ai]||(o[Ai]=!0,gc("selectionchange",!1,o))}}function _f(e,o,l,d){switch(Zd(o)){case 1:var m=fx;break;case 4:m=px;break;default:m=Zl}l=m.bind(null,o,l,e),m=void 0,!En||o!=="touchstart"&&o!=="touchmove"&&o!=="wheel"||(m=!0),d?m!==void 0?e.addEventListener(o,l,{capture:!0,passive:m}):e.addEventListener(o,l,!0):m!==void 0?e.addEventListener(o,l,{passive:m}):e.addEventListener(o,l,!1)}function xc(e,o,l,d,m){var v=d;if((o&1)===0&&(o&2)===0&&d!==null)e:for(;;){if(d===null)return;var R=d.tag;if(R===3||R===4){var j=d.stateNode.containerInfo;if(j===m||j.nodeType===8&&j.parentNode===m)break;if(R===4)for(R=d.return;R!==null;){var U=R.tag;if((U===3||U===4)&&(U=R.stateNode.containerInfo,U===m||U.nodeType===8&&U.parentNode===m))return;R=R.return}for(;j!==null;){if(R=ao(j),R===null)return;if(U=R.tag,U===5||U===6){d=v=R;continue e}j=j.parentNode}}d=d.return}wn(function(){var J=v,le=vn(l),ce=[];e:{var ie=Pf.get(e);if(ie!==void 0){var we=rc,Re=e;switch(e){case"keypress":if(wi(l)===0)break e;case"keydown":case"keyup":we=kx;break;case"focusin":Re="focus",we=ac;break;case"focusout":Re="blur",we=ac;break;case"beforeblur":case"afterblur":we=ac;break;case"click":if(l.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":we=rf;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":we=gx;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":we=_x;break;case bf:case Sf:case Rf:we=yx;break;case Af:we=Fx;break;case"scroll":we=hx;break;case"wheel":we=Ix;break;case"copy":case"cut":case"paste":we=Ex;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":we=of}var Pe=(o&4)!==0,Pt=!Pe&&e==="scroll",q=Pe?ie!==null?ie+"Capture":null:ie;Pe=[];for(var V=J,X;V!==null;){X=V;var de=X.stateNode;if(X.tag===5&&de!==null&&(X=de,q!==null&&(de=Xr(V,q),de!=null&&Pe.push(Ca(V,de,X)))),Pt)break;V=V.return}0<Pe.length&&(ie=new we(ie,Re,null,l,le),ce.push({event:ie,listeners:Pe}))}}if((o&7)===0){e:{if(ie=e==="mouseover"||e==="pointerover",we=e==="mouseout"||e==="pointerout",ie&&l!==xn&&(Re=l.relatedTarget||l.fromElement)&&(ao(Re)||Re[Zr]))break e;if((we||ie)&&(ie=le.window===le?le:(ie=le.ownerDocument)?ie.defaultView||ie.parentWindow:window,we?(Re=l.relatedTarget||l.toElement,we=J,Re=Re?ao(Re):null,Re!==null&&(Pt=Ae(Re),Re!==Pt||Re.tag!==5&&Re.tag!==6)&&(Re=null)):(we=null,Re=J),we!==Re)){if(Pe=rf,de="onMouseLeave",q="onMouseEnter",V="mouse",(e==="pointerout"||e==="pointerover")&&(Pe=of,de="onPointerLeave",q="onPointerEnter",V="pointer"),Pt=we==null?ie:jo(we),X=Re==null?ie:jo(Re),ie=new Pe(de,V+"leave",we,l,le),ie.target=Pt,ie.relatedTarget=X,de=null,ao(le)===J&&(Pe=new Pe(q,V+"enter",Re,l,le),Pe.target=X,Pe.relatedTarget=Pt,de=Pe),Pt=de,we&&Re)t:{for(Pe=we,q=Re,V=0,X=Pe;X;X=Io(X))V++;for(X=0,de=q;de;de=Io(de))X++;for(;0<V-X;)Pe=Io(Pe),V--;for(;0<X-V;)q=Io(q),X--;for(;V--;){if(Pe===q||q!==null&&Pe===q.alternate)break t;Pe=Io(Pe),q=Io(q)}Pe=null}else Pe=null;we!==null&&Lf(ce,ie,we,Pe,!1),Re!==null&&Pt!==null&&Lf(ce,Pt,Re,Pe,!0)}}e:{if(ie=J?jo(J):window,we=ie.nodeName&&ie.nodeName.toLowerCase(),we==="select"||we==="input"&&ie.type==="file")var Te=zx;else if(df(ie))if(pf)Te=Gx;else{Te=Vx;var Ie=Hx}else(we=ie.nodeName)&&we.toLowerCase()==="input"&&(ie.type==="checkbox"||ie.type==="radio")&&(Te=Wx);if(Te&&(Te=Te(e,J))){ff(ce,Te,l,le);break e}Ie&&Ie(e,ie,J),e==="focusout"&&(Ie=ie._wrapperState)&&Ie.controlled&&ie.type==="number"&&ke(ie,"number",ie.value)}switch(Ie=J?jo(J):window,e){case"focusin":(df(Ie)||Ie.contentEditable==="true")&&(Fo=Ie,dc=J,ya=null);break;case"focusout":ya=dc=Fo=null;break;case"mousedown":fc=!0;break;case"contextmenu":case"mouseup":case"dragend":fc=!1,Ef(ce,l,le);break;case"selectionchange":if(Xx)break;case"keydown":case"keyup":Ef(ce,l,le)}var Be;if(sc)e:{switch(e){case"compositionstart":var Me="onCompositionStart";break e;case"compositionend":Me="onCompositionEnd";break e;case"compositionupdate":Me="onCompositionUpdate";break e}Me=void 0}else Lo?cf(e,l)&&(Me="onCompositionEnd"):e==="keydown"&&l.keyCode===229&&(Me="onCompositionStart");Me&&(af&&l.locale!=="ko"&&(Lo||Me!=="onCompositionStart"?Me==="onCompositionEnd"&&Lo&&(Be=ef()):(kn=le,tc="value"in kn?kn.value:kn.textContent,Lo=!0)),Ie=Pi(J,Me),0<Ie.length&&(Me=new nf(Me,e,null,l,le),ce.push({event:Me,listeners:Ie}),Be?Me.data=Be:(Be=uf(l),Be!==null&&(Me.data=Be)))),(Be=jx?Ox(e,l):Mx(e,l))&&(J=Pi(J,"onBeforeInput"),0<J.length&&(le=new nf("onBeforeInput","beforeinput",null,l,le),ce.push({event:le,listeners:J}),le.data=Be))}Tf(ce,o)})}function Ca(e,o,l){return{instance:e,listener:o,currentTarget:l}}function Pi(e,o){for(var l=o+"Capture",d=[];e!==null;){var m=e,v=m.stateNode;m.tag===5&&v!==null&&(m=v,v=Xr(e,l),v!=null&&d.unshift(Ca(e,v,m)),v=Xr(e,o),v!=null&&d.push(Ca(e,v,m))),e=e.return}return d}function Io(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function Lf(e,o,l,d,m){for(var v=o._reactName,R=[];l!==null&&l!==d;){var j=l,U=j.alternate,J=j.stateNode;if(U!==null&&U===d)break;j.tag===5&&J!==null&&(j=J,m?(U=Xr(l,v),U!=null&&R.unshift(Ca(l,U,j))):m||(U=Xr(l,v),U!=null&&R.push(Ca(l,U,j)))),l=l.return}R.length!==0&&e.push({event:o,listeners:R})}var Zx=/\r\n?/g,ev=/\u0000|\uFFFD/g;function Ff(e){return(typeof e=="string"?e:""+e).replace(Zx,`
`).replace(ev,"")}function ki(e,o,l){if(o=Ff(o),Ff(e)!==o&&l)throw Error(n(425))}function Di(){}var vc=null,yc=null;function wc(e,o){return e==="textarea"||e==="noscript"||typeof o.children=="string"||typeof o.children=="number"||typeof o.dangerouslySetInnerHTML=="object"&&o.dangerouslySetInnerHTML!==null&&o.dangerouslySetInnerHTML.__html!=null}var Ec=typeof setTimeout=="function"?setTimeout:void 0,tv=typeof clearTimeout=="function"?clearTimeout:void 0,Nf=typeof Promise=="function"?Promise:void 0,rv=typeof queueMicrotask=="function"?queueMicrotask:typeof Nf<"u"?function(e){return Nf.resolve(null).then(e).catch(nv)}:Ec;function nv(e){setTimeout(function(){throw e})}function Cc(e,o){var l=o,d=0;do{var m=l.nextSibling;if(e.removeChild(l),m&&m.nodeType===8)if(l=m.data,l==="/$"){if(d===0){e.removeChild(m),fa(o);return}d--}else l!=="$"&&l!=="$?"&&l!=="$!"||d++;l=m}while(l);fa(o)}function Tn(e){for(;e!=null;e=e.nextSibling){var o=e.nodeType;if(o===1||o===3)break;if(o===8){if(o=e.data,o==="$"||o==="$!"||o==="$?")break;if(o==="/$")return null}}return e}function If(e){e=e.previousSibling;for(var o=0;e;){if(e.nodeType===8){var l=e.data;if(l==="$"||l==="$!"||l==="$?"){if(o===0)return e;o--}else l==="/$"&&o++}e=e.previousSibling}return null}var Bo=Math.random().toString(36).slice(2),$r="__reactFiber$"+Bo,ba="__reactProps$"+Bo,Zr="__reactContainer$"+Bo,bc="__reactEvents$"+Bo,ov="__reactListeners$"+Bo,av="__reactHandles$"+Bo;function ao(e){var o=e[$r];if(o)return o;for(var l=e.parentNode;l;){if(o=l[Zr]||l[$r]){if(l=o.alternate,o.child!==null||l!==null&&l.child!==null)for(e=If(e);e!==null;){if(l=e[$r])return l;e=If(e)}return o}e=l,l=e.parentNode}return null}function Sa(e){return e=e[$r]||e[Zr],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function jo(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(n(33))}function Ti(e){return e[ba]||null}var Sc=[],Oo=-1;function _n(e){return{current:e}}function xt(e){0>Oo||(e.current=Sc[Oo],Sc[Oo]=null,Oo--)}function pt(e,o){Oo++,Sc[Oo]=e.current,e.current=o}var Ln={},Wt=_n(Ln),rr=_n(!1),io=Ln;function Mo(e,o){var l=e.type.contextTypes;if(!l)return Ln;var d=e.stateNode;if(d&&d.__reactInternalMemoizedUnmaskedChildContext===o)return d.__reactInternalMemoizedMaskedChildContext;var m={},v;for(v in l)m[v]=o[v];return d&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=o,e.__reactInternalMemoizedMaskedChildContext=m),m}function nr(e){return e=e.childContextTypes,e!=null}function _i(){xt(rr),xt(Wt)}function Bf(e,o,l){if(Wt.current!==Ln)throw Error(n(168));pt(Wt,o),pt(rr,l)}function jf(e,o,l){var d=e.stateNode;if(o=o.childContextTypes,typeof d.getChildContext!="function")return l;d=d.getChildContext();for(var m in d)if(!(m in o))throw Error(n(108,Le(e)||"Unknown",m));return ee({},l,d)}function Li(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||Ln,io=Wt.current,pt(Wt,e),pt(rr,rr.current),!0}function Of(e,o,l){var d=e.stateNode;if(!d)throw Error(n(169));l?(e=jf(e,o,io),d.__reactInternalMemoizedMergedChildContext=e,xt(rr),xt(Wt),pt(Wt,e)):xt(rr),pt(rr,l)}var en=null,Fi=!1,Rc=!1;function Mf(e){en===null?en=[e]:en.push(e)}function iv(e){Fi=!0,Mf(e)}function Fn(){if(!Rc&&en!==null){Rc=!0;var e=0,o=it;try{var l=en;for(it=1;e<l.length;e++){var d=l[e];do d=d(!0);while(d!==null)}en=null,Fi=!1}catch(m){throw en!==null&&(en=en.slice(e+1)),At(xr,Fn),m}finally{it=o,Rc=!1}}return null}var $o=[],Uo=0,Ni=null,Ii=0,vr=[],yr=0,so=null,tn=1,rn="";function lo(e,o){$o[Uo++]=Ii,$o[Uo++]=Ni,Ni=e,Ii=o}function $f(e,o,l){vr[yr++]=tn,vr[yr++]=rn,vr[yr++]=so,so=e;var d=tn;e=rn;var m=32-ut(d)-1;d&=~(1<<m),l+=1;var v=32-ut(o)+m;if(30<v){var R=m-m%5;v=(d&(1<<R)-1).toString(32),d>>=R,m-=R,tn=1<<32-ut(o)+m|l<<m|d,rn=v+e}else tn=1<<v|l<<m|d,rn=e}function Ac(e){e.return!==null&&(lo(e,1),$f(e,1,0))}function Pc(e){for(;e===Ni;)Ni=$o[--Uo],$o[Uo]=null,Ii=$o[--Uo],$o[Uo]=null;for(;e===so;)so=vr[--yr],vr[yr]=null,rn=vr[--yr],vr[yr]=null,tn=vr[--yr],vr[yr]=null}var pr=null,hr=null,yt=!1,Tr=null;function Uf(e,o){var l=br(5,null,null,0);l.elementType="DELETED",l.stateNode=o,l.return=e,o=e.deletions,o===null?(e.deletions=[l],e.flags|=16):o.push(l)}function zf(e,o){switch(e.tag){case 5:var l=e.type;return o=o.nodeType!==1||l.toLowerCase()!==o.nodeName.toLowerCase()?null:o,o!==null?(e.stateNode=o,pr=e,hr=Tn(o.firstChild),!0):!1;case 6:return o=e.pendingProps===""||o.nodeType!==3?null:o,o!==null?(e.stateNode=o,pr=e,hr=null,!0):!1;case 13:return o=o.nodeType!==8?null:o,o!==null?(l=so!==null?{id:tn,overflow:rn}:null,e.memoizedState={dehydrated:o,treeContext:l,retryLane:1073741824},l=br(18,null,null,0),l.stateNode=o,l.return=e,e.child=l,pr=e,hr=null,!0):!1;default:return!1}}function kc(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Dc(e){if(yt){var o=hr;if(o){var l=o;if(!zf(e,o)){if(kc(e))throw Error(n(418));o=Tn(l.nextSibling);var d=pr;o&&zf(e,o)?Uf(d,l):(e.flags=e.flags&-4097|2,yt=!1,pr=e)}}else{if(kc(e))throw Error(n(418));e.flags=e.flags&-4097|2,yt=!1,pr=e}}}function Hf(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;pr=e}function Bi(e){if(e!==pr)return!1;if(!yt)return Hf(e),yt=!0,!1;var o;if((o=e.tag!==3)&&!(o=e.tag!==5)&&(o=e.type,o=o!=="head"&&o!=="body"&&!wc(e.type,e.memoizedProps)),o&&(o=hr)){if(kc(e))throw Vf(),Error(n(418));for(;o;)Uf(e,o),o=Tn(o.nextSibling)}if(Hf(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(n(317));e:{for(e=e.nextSibling,o=0;e;){if(e.nodeType===8){var l=e.data;if(l==="/$"){if(o===0){hr=Tn(e.nextSibling);break e}o--}else l!=="$"&&l!=="$!"&&l!=="$?"||o++}e=e.nextSibling}hr=null}}else hr=pr?Tn(e.stateNode.nextSibling):null;return!0}function Vf(){for(var e=hr;e;)e=Tn(e.nextSibling)}function zo(){hr=pr=null,yt=!1}function Tc(e){Tr===null?Tr=[e]:Tr.push(e)}var sv=T.ReactCurrentBatchConfig;function Ra(e,o,l){if(e=l.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(l._owner){if(l=l._owner,l){if(l.tag!==1)throw Error(n(309));var d=l.stateNode}if(!d)throw Error(n(147,e));var m=d,v=""+e;return o!==null&&o.ref!==null&&typeof o.ref=="function"&&o.ref._stringRef===v?o.ref:(o=function(R){var j=m.refs;R===null?delete j[v]:j[v]=R},o._stringRef=v,o)}if(typeof e!="string")throw Error(n(284));if(!l._owner)throw Error(n(290,e))}return e}function ji(e,o){throw e=Object.prototype.toString.call(o),Error(n(31,e==="[object Object]"?"object with keys {"+Object.keys(o).join(", ")+"}":e))}function Wf(e){var o=e._init;return o(e._payload)}function Gf(e){function o(q,V){if(e){var X=q.deletions;X===null?(q.deletions=[V],q.flags|=16):X.push(V)}}function l(q,V){if(!e)return null;for(;V!==null;)o(q,V),V=V.sibling;return null}function d(q,V){for(q=new Map;V!==null;)V.key!==null?q.set(V.key,V):q.set(V.index,V),V=V.sibling;return q}function m(q,V){return q=Un(q,V),q.index=0,q.sibling=null,q}function v(q,V,X){return q.index=X,e?(X=q.alternate,X!==null?(X=X.index,X<V?(q.flags|=2,V):X):(q.flags|=2,V)):(q.flags|=1048576,V)}function R(q){return e&&q.alternate===null&&(q.flags|=2),q}function j(q,V,X,de){return V===null||V.tag!==6?(V=Eu(X,q.mode,de),V.return=q,V):(V=m(V,X),V.return=q,V)}function U(q,V,X,de){var Te=X.type;return Te===k?le(q,V,X.props.children,de,X.key):V!==null&&(V.elementType===Te||typeof Te=="object"&&Te!==null&&Te.$$typeof===re&&Wf(Te)===V.type)?(de=m(V,X.props),de.ref=Ra(q,V,X),de.return=q,de):(de=ss(X.type,X.key,X.props,null,q.mode,de),de.ref=Ra(q,V,X),de.return=q,de)}function J(q,V,X,de){return V===null||V.tag!==4||V.stateNode.containerInfo!==X.containerInfo||V.stateNode.implementation!==X.implementation?(V=Cu(X,q.mode,de),V.return=q,V):(V=m(V,X.children||[]),V.return=q,V)}function le(q,V,X,de,Te){return V===null||V.tag!==7?(V=xo(X,q.mode,de,Te),V.return=q,V):(V=m(V,X),V.return=q,V)}function ce(q,V,X){if(typeof V=="string"&&V!==""||typeof V=="number")return V=Eu(""+V,q.mode,X),V.return=q,V;if(typeof V=="object"&&V!==null){switch(V.$$typeof){case B:return X=ss(V.type,V.key,V.props,null,q.mode,X),X.ref=Ra(q,null,V),X.return=q,X;case F:return V=Cu(V,q.mode,X),V.return=q,V;case re:var de=V._init;return ce(q,de(V._payload),X)}if(xe(V)||te(V))return V=xo(V,q.mode,X,null),V.return=q,V;ji(q,V)}return null}function ie(q,V,X,de){var Te=V!==null?V.key:null;if(typeof X=="string"&&X!==""||typeof X=="number")return Te!==null?null:j(q,V,""+X,de);if(typeof X=="object"&&X!==null){switch(X.$$typeof){case B:return X.key===Te?U(q,V,X,de):null;case F:return X.key===Te?J(q,V,X,de):null;case re:return Te=X._init,ie(q,V,Te(X._payload),de)}if(xe(X)||te(X))return Te!==null?null:le(q,V,X,de,null);ji(q,X)}return null}function we(q,V,X,de,Te){if(typeof de=="string"&&de!==""||typeof de=="number")return q=q.get(X)||null,j(V,q,""+de,Te);if(typeof de=="object"&&de!==null){switch(de.$$typeof){case B:return q=q.get(de.key===null?X:de.key)||null,U(V,q,de,Te);case F:return q=q.get(de.key===null?X:de.key)||null,J(V,q,de,Te);case re:var Ie=de._init;return we(q,V,X,Ie(de._payload),Te)}if(xe(de)||te(de))return q=q.get(X)||null,le(V,q,de,Te,null);ji(V,de)}return null}function Re(q,V,X,de){for(var Te=null,Ie=null,Be=V,Me=V=0,Mt=null;Be!==null&&Me<X.length;Me++){Be.index>Me?(Mt=Be,Be=null):Mt=Be.sibling;var et=ie(q,Be,X[Me],de);if(et===null){Be===null&&(Be=Mt);break}e&&Be&&et.alternate===null&&o(q,Be),V=v(et,V,Me),Ie===null?Te=et:Ie.sibling=et,Ie=et,Be=Mt}if(Me===X.length)return l(q,Be),yt&&lo(q,Me),Te;if(Be===null){for(;Me<X.length;Me++)Be=ce(q,X[Me],de),Be!==null&&(V=v(Be,V,Me),Ie===null?Te=Be:Ie.sibling=Be,Ie=Be);return yt&&lo(q,Me),Te}for(Be=d(q,Be);Me<X.length;Me++)Mt=we(Be,q,Me,X[Me],de),Mt!==null&&(e&&Mt.alternate!==null&&Be.delete(Mt.key===null?Me:Mt.key),V=v(Mt,V,Me),Ie===null?Te=Mt:Ie.sibling=Mt,Ie=Mt);return e&&Be.forEach(function(zn){return o(q,zn)}),yt&&lo(q,Me),Te}function Pe(q,V,X,de){var Te=te(X);if(typeof Te!="function")throw Error(n(150));if(X=Te.call(X),X==null)throw Error(n(151));for(var Ie=Te=null,Be=V,Me=V=0,Mt=null,et=X.next();Be!==null&&!et.done;Me++,et=X.next()){Be.index>Me?(Mt=Be,Be=null):Mt=Be.sibling;var zn=ie(q,Be,et.value,de);if(zn===null){Be===null&&(Be=Mt);break}e&&Be&&zn.alternate===null&&o(q,Be),V=v(zn,V,Me),Ie===null?Te=zn:Ie.sibling=zn,Ie=zn,Be=Mt}if(et.done)return l(q,Be),yt&&lo(q,Me),Te;if(Be===null){for(;!et.done;Me++,et=X.next())et=ce(q,et.value,de),et!==null&&(V=v(et,V,Me),Ie===null?Te=et:Ie.sibling=et,Ie=et);return yt&&lo(q,Me),Te}for(Be=d(q,Be);!et.done;Me++,et=X.next())et=we(Be,q,Me,et.value,de),et!==null&&(e&&et.alternate!==null&&Be.delete(et.key===null?Me:et.key),V=v(et,V,Me),Ie===null?Te=et:Ie.sibling=et,Ie=et);return e&&Be.forEach(function($v){return o(q,$v)}),yt&&lo(q,Me),Te}function Pt(q,V,X,de){if(typeof X=="object"&&X!==null&&X.type===k&&X.key===null&&(X=X.props.children),typeof X=="object"&&X!==null){switch(X.$$typeof){case B:e:{for(var Te=X.key,Ie=V;Ie!==null;){if(Ie.key===Te){if(Te=X.type,Te===k){if(Ie.tag===7){l(q,Ie.sibling),V=m(Ie,X.props.children),V.return=q,q=V;break e}}else if(Ie.elementType===Te||typeof Te=="object"&&Te!==null&&Te.$$typeof===re&&Wf(Te)===Ie.type){l(q,Ie.sibling),V=m(Ie,X.props),V.ref=Ra(q,Ie,X),V.return=q,q=V;break e}l(q,Ie);break}else o(q,Ie);Ie=Ie.sibling}X.type===k?(V=xo(X.props.children,q.mode,de,X.key),V.return=q,q=V):(de=ss(X.type,X.key,X.props,null,q.mode,de),de.ref=Ra(q,V,X),de.return=q,q=de)}return R(q);case F:e:{for(Ie=X.key;V!==null;){if(V.key===Ie)if(V.tag===4&&V.stateNode.containerInfo===X.containerInfo&&V.stateNode.implementation===X.implementation){l(q,V.sibling),V=m(V,X.children||[]),V.return=q,q=V;break e}else{l(q,V);break}else o(q,V);V=V.sibling}V=Cu(X,q.mode,de),V.return=q,q=V}return R(q);case re:return Ie=X._init,Pt(q,V,Ie(X._payload),de)}if(xe(X))return Re(q,V,X,de);if(te(X))return Pe(q,V,X,de);ji(q,X)}return typeof X=="string"&&X!==""||typeof X=="number"?(X=""+X,V!==null&&V.tag===6?(l(q,V.sibling),V=m(V,X),V.return=q,q=V):(l(q,V),V=Eu(X,q.mode,de),V.return=q,q=V),R(q)):l(q,V)}return Pt}var Ho=Gf(!0),qf=Gf(!1),Oi=_n(null),Mi=null,Vo=null,_c=null;function Lc(){_c=Vo=Mi=null}function Fc(e){var o=Oi.current;xt(Oi),e._currentValue=o}function Nc(e,o,l){for(;e!==null;){var d=e.alternate;if((e.childLanes&o)!==o?(e.childLanes|=o,d!==null&&(d.childLanes|=o)):d!==null&&(d.childLanes&o)!==o&&(d.childLanes|=o),e===l)break;e=e.return}}function Wo(e,o){Mi=e,_c=Vo=null,e=e.dependencies,e!==null&&e.firstContext!==null&&((e.lanes&o)!==0&&(or=!0),e.firstContext=null)}function wr(e){var o=e._currentValue;if(_c!==e)if(e={context:e,memoizedValue:o,next:null},Vo===null){if(Mi===null)throw Error(n(308));Vo=e,Mi.dependencies={lanes:0,firstContext:e}}else Vo=Vo.next=e;return o}var co=null;function Ic(e){co===null?co=[e]:co.push(e)}function Kf(e,o,l,d){var m=o.interleaved;return m===null?(l.next=l,Ic(o)):(l.next=m.next,m.next=l),o.interleaved=l,nn(e,d)}function nn(e,o){e.lanes|=o;var l=e.alternate;for(l!==null&&(l.lanes|=o),l=e,e=e.return;e!==null;)e.childLanes|=o,l=e.alternate,l!==null&&(l.childLanes|=o),l=e,e=e.return;return l.tag===3?l.stateNode:null}var Nn=!1;function Bc(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function Xf(e,o){e=e.updateQueue,o.updateQueue===e&&(o.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function on(e,o){return{eventTime:e,lane:o,tag:0,payload:null,callback:null,next:null}}function In(e,o,l){var d=e.updateQueue;if(d===null)return null;if(d=d.shared,(Ze&2)!==0){var m=d.pending;return m===null?o.next=o:(o.next=m.next,m.next=o),d.pending=o,nn(e,l)}return m=d.interleaved,m===null?(o.next=o,Ic(d)):(o.next=m.next,m.next=o),d.interleaved=o,nn(e,l)}function $i(e,o,l){if(o=o.updateQueue,o!==null&&(o=o.shared,(l&4194240)!==0)){var d=o.lanes;d&=e.pendingLanes,l|=d,o.lanes=l,Yl(e,l)}}function Yf(e,o){var l=e.updateQueue,d=e.alternate;if(d!==null&&(d=d.updateQueue,l===d)){var m=null,v=null;if(l=l.firstBaseUpdate,l!==null){do{var R={eventTime:l.eventTime,lane:l.lane,tag:l.tag,payload:l.payload,callback:l.callback,next:null};v===null?m=v=R:v=v.next=R,l=l.next}while(l!==null);v===null?m=v=o:v=v.next=o}else m=v=o;l={baseState:d.baseState,firstBaseUpdate:m,lastBaseUpdate:v,shared:d.shared,effects:d.effects},e.updateQueue=l;return}e=l.lastBaseUpdate,e===null?l.firstBaseUpdate=o:e.next=o,l.lastBaseUpdate=o}function Ui(e,o,l,d){var m=e.updateQueue;Nn=!1;var v=m.firstBaseUpdate,R=m.lastBaseUpdate,j=m.shared.pending;if(j!==null){m.shared.pending=null;var U=j,J=U.next;U.next=null,R===null?v=J:R.next=J,R=U;var le=e.alternate;le!==null&&(le=le.updateQueue,j=le.lastBaseUpdate,j!==R&&(j===null?le.firstBaseUpdate=J:j.next=J,le.lastBaseUpdate=U))}if(v!==null){var ce=m.baseState;R=0,le=J=U=null,j=v;do{var ie=j.lane,we=j.eventTime;if((d&ie)===ie){le!==null&&(le=le.next={eventTime:we,lane:0,tag:j.tag,payload:j.payload,callback:j.callback,next:null});e:{var Re=e,Pe=j;switch(ie=o,we=l,Pe.tag){case 1:if(Re=Pe.payload,typeof Re=="function"){ce=Re.call(we,ce,ie);break e}ce=Re;break e;case 3:Re.flags=Re.flags&-65537|128;case 0:if(Re=Pe.payload,ie=typeof Re=="function"?Re.call(we,ce,ie):Re,ie==null)break e;ce=ee({},ce,ie);break e;case 2:Nn=!0}}j.callback!==null&&j.lane!==0&&(e.flags|=64,ie=m.effects,ie===null?m.effects=[j]:ie.push(j))}else we={eventTime:we,lane:ie,tag:j.tag,payload:j.payload,callback:j.callback,next:null},le===null?(J=le=we,U=ce):le=le.next=we,R|=ie;if(j=j.next,j===null){if(j=m.shared.pending,j===null)break;ie=j,j=ie.next,ie.next=null,m.lastBaseUpdate=ie,m.shared.pending=null}}while(!0);if(le===null&&(U=ce),m.baseState=U,m.firstBaseUpdate=J,m.lastBaseUpdate=le,o=m.shared.interleaved,o!==null){m=o;do R|=m.lane,m=m.next;while(m!==o)}else v===null&&(m.shared.lanes=0);po|=R,e.lanes=R,e.memoizedState=ce}}function Qf(e,o,l){if(e=o.effects,o.effects=null,e!==null)for(o=0;o<e.length;o++){var d=e[o],m=d.callback;if(m!==null){if(d.callback=null,d=l,typeof m!="function")throw Error(n(191,m));m.call(d)}}}var Aa={},Ur=_n(Aa),Pa=_n(Aa),ka=_n(Aa);function uo(e){if(e===Aa)throw Error(n(174));return e}function jc(e,o){switch(pt(ka,o),pt(Pa,e),pt(Ur,Aa),e=o.nodeType,e){case 9:case 11:o=(o=o.documentElement)?o.namespaceURI:Rt(null,"");break;default:e=e===8?o.parentNode:o,o=e.namespaceURI||null,e=e.tagName,o=Rt(o,e)}xt(Ur),pt(Ur,o)}function Go(){xt(Ur),xt(Pa),xt(ka)}function Jf(e){uo(ka.current);var o=uo(Ur.current),l=Rt(o,e.type);o!==l&&(pt(Pa,e),pt(Ur,l))}function Oc(e){Pa.current===e&&(xt(Ur),xt(Pa))}var wt=_n(0);function zi(e){for(var o=e;o!==null;){if(o.tag===13){var l=o.memoizedState;if(l!==null&&(l=l.dehydrated,l===null||l.data==="$?"||l.data==="$!"))return o}else if(o.tag===19&&o.memoizedProps.revealOrder!==void 0){if((o.flags&128)!==0)return o}else if(o.child!==null){o.child.return=o,o=o.child;continue}if(o===e)break;for(;o.sibling===null;){if(o.return===null||o.return===e)return null;o=o.return}o.sibling.return=o.return,o=o.sibling}return null}var Mc=[];function $c(){for(var e=0;e<Mc.length;e++)Mc[e]._workInProgressVersionPrimary=null;Mc.length=0}var Hi=T.ReactCurrentDispatcher,Uc=T.ReactCurrentBatchConfig,fo=0,Et=null,Nt=null,jt=null,Vi=!1,Da=!1,Ta=0,lv=0;function Gt(){throw Error(n(321))}function zc(e,o){if(o===null)return!1;for(var l=0;l<o.length&&l<e.length;l++)if(!Dr(e[l],o[l]))return!1;return!0}function Hc(e,o,l,d,m,v){if(fo=v,Et=o,o.memoizedState=null,o.updateQueue=null,o.lanes=0,Hi.current=e===null||e.memoizedState===null?fv:pv,e=l(d,m),Da){v=0;do{if(Da=!1,Ta=0,25<=v)throw Error(n(301));v+=1,jt=Nt=null,o.updateQueue=null,Hi.current=hv,e=l(d,m)}while(Da)}if(Hi.current=qi,o=Nt!==null&&Nt.next!==null,fo=0,jt=Nt=Et=null,Vi=!1,o)throw Error(n(300));return e}function Vc(){var e=Ta!==0;return Ta=0,e}function zr(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return jt===null?Et.memoizedState=jt=e:jt=jt.next=e,jt}function Er(){if(Nt===null){var e=Et.alternate;e=e!==null?e.memoizedState:null}else e=Nt.next;var o=jt===null?Et.memoizedState:jt.next;if(o!==null)jt=o,Nt=e;else{if(e===null)throw Error(n(310));Nt=e,e={memoizedState:Nt.memoizedState,baseState:Nt.baseState,baseQueue:Nt.baseQueue,queue:Nt.queue,next:null},jt===null?Et.memoizedState=jt=e:jt=jt.next=e}return jt}function _a(e,o){return typeof o=="function"?o(e):o}function Wc(e){var o=Er(),l=o.queue;if(l===null)throw Error(n(311));l.lastRenderedReducer=e;var d=Nt,m=d.baseQueue,v=l.pending;if(v!==null){if(m!==null){var R=m.next;m.next=v.next,v.next=R}d.baseQueue=m=v,l.pending=null}if(m!==null){v=m.next,d=d.baseState;var j=R=null,U=null,J=v;do{var le=J.lane;if((fo&le)===le)U!==null&&(U=U.next={lane:0,action:J.action,hasEagerState:J.hasEagerState,eagerState:J.eagerState,next:null}),d=J.hasEagerState?J.eagerState:e(d,J.action);else{var ce={lane:le,action:J.action,hasEagerState:J.hasEagerState,eagerState:J.eagerState,next:null};U===null?(j=U=ce,R=d):U=U.next=ce,Et.lanes|=le,po|=le}J=J.next}while(J!==null&&J!==v);U===null?R=d:U.next=j,Dr(d,o.memoizedState)||(or=!0),o.memoizedState=d,o.baseState=R,o.baseQueue=U,l.lastRenderedState=d}if(e=l.interleaved,e!==null){m=e;do v=m.lane,Et.lanes|=v,po|=v,m=m.next;while(m!==e)}else m===null&&(l.lanes=0);return[o.memoizedState,l.dispatch]}function Gc(e){var o=Er(),l=o.queue;if(l===null)throw Error(n(311));l.lastRenderedReducer=e;var d=l.dispatch,m=l.pending,v=o.memoizedState;if(m!==null){l.pending=null;var R=m=m.next;do v=e(v,R.action),R=R.next;while(R!==m);Dr(v,o.memoizedState)||(or=!0),o.memoizedState=v,o.baseQueue===null&&(o.baseState=v),l.lastRenderedState=v}return[v,d]}function Zf(){}function e0(e,o){var l=Et,d=Er(),m=o(),v=!Dr(d.memoizedState,m);if(v&&(d.memoizedState=m,or=!0),d=d.queue,qc(n0.bind(null,l,d,e),[e]),d.getSnapshot!==o||v||jt!==null&&jt.memoizedState.tag&1){if(l.flags|=2048,La(9,r0.bind(null,l,d,m,o),void 0,null),Ot===null)throw Error(n(349));(fo&30)!==0||t0(l,o,m)}return m}function t0(e,o,l){e.flags|=16384,e={getSnapshot:o,value:l},o=Et.updateQueue,o===null?(o={lastEffect:null,stores:null},Et.updateQueue=o,o.stores=[e]):(l=o.stores,l===null?o.stores=[e]:l.push(e))}function r0(e,o,l,d){o.value=l,o.getSnapshot=d,o0(o)&&a0(e)}function n0(e,o,l){return l(function(){o0(o)&&a0(e)})}function o0(e){var o=e.getSnapshot;e=e.value;try{var l=o();return!Dr(e,l)}catch{return!0}}function a0(e){var o=nn(e,1);o!==null&&Nr(o,e,1,-1)}function i0(e){var o=zr();return typeof e=="function"&&(e=e()),o.memoizedState=o.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:_a,lastRenderedState:e},o.queue=e,e=e.dispatch=dv.bind(null,Et,e),[o.memoizedState,e]}function La(e,o,l,d){return e={tag:e,create:o,destroy:l,deps:d,next:null},o=Et.updateQueue,o===null?(o={lastEffect:null,stores:null},Et.updateQueue=o,o.lastEffect=e.next=e):(l=o.lastEffect,l===null?o.lastEffect=e.next=e:(d=l.next,l.next=e,e.next=d,o.lastEffect=e)),e}function s0(){return Er().memoizedState}function Wi(e,o,l,d){var m=zr();Et.flags|=e,m.memoizedState=La(1|o,l,void 0,d===void 0?null:d)}function Gi(e,o,l,d){var m=Er();d=d===void 0?null:d;var v=void 0;if(Nt!==null){var R=Nt.memoizedState;if(v=R.destroy,d!==null&&zc(d,R.deps)){m.memoizedState=La(o,l,v,d);return}}Et.flags|=e,m.memoizedState=La(1|o,l,v,d)}function l0(e,o){return Wi(8390656,8,e,o)}function qc(e,o){return Gi(2048,8,e,o)}function c0(e,o){return Gi(4,2,e,o)}function u0(e,o){return Gi(4,4,e,o)}function d0(e,o){if(typeof o=="function")return e=e(),o(e),function(){o(null)};if(o!=null)return e=e(),o.current=e,function(){o.current=null}}function f0(e,o,l){return l=l!=null?l.concat([e]):null,Gi(4,4,d0.bind(null,o,e),l)}function Kc(){}function p0(e,o){var l=Er();o=o===void 0?null:o;var d=l.memoizedState;return d!==null&&o!==null&&zc(o,d[1])?d[0]:(l.memoizedState=[e,o],e)}function h0(e,o){var l=Er();o=o===void 0?null:o;var d=l.memoizedState;return d!==null&&o!==null&&zc(o,d[1])?d[0]:(e=e(),l.memoizedState=[e,o],e)}function m0(e,o,l){return(fo&21)===0?(e.baseState&&(e.baseState=!1,or=!0),e.memoizedState=l):(Dr(l,o)||(l=Vd(),Et.lanes|=l,po|=l,e.baseState=!0),o)}function cv(e,o){var l=it;it=l!==0&&4>l?l:4,e(!0);var d=Uc.transition;Uc.transition={};try{e(!1),o()}finally{it=l,Uc.transition=d}}function g0(){return Er().memoizedState}function uv(e,o,l){var d=Mn(e);if(l={lane:d,action:l,hasEagerState:!1,eagerState:null,next:null},x0(e))v0(o,l);else if(l=Kf(e,o,l,d),l!==null){var m=Zt();Nr(l,e,d,m),y0(l,o,d)}}function dv(e,o,l){var d=Mn(e),m={lane:d,action:l,hasEagerState:!1,eagerState:null,next:null};if(x0(e))v0(o,m);else{var v=e.alternate;if(e.lanes===0&&(v===null||v.lanes===0)&&(v=o.lastRenderedReducer,v!==null))try{var R=o.lastRenderedState,j=v(R,l);if(m.hasEagerState=!0,m.eagerState=j,Dr(j,R)){var U=o.interleaved;U===null?(m.next=m,Ic(o)):(m.next=U.next,U.next=m),o.interleaved=m;return}}catch{}finally{}l=Kf(e,o,m,d),l!==null&&(m=Zt(),Nr(l,e,d,m),y0(l,o,d))}}function x0(e){var o=e.alternate;return e===Et||o!==null&&o===Et}function v0(e,o){Da=Vi=!0;var l=e.pending;l===null?o.next=o:(o.next=l.next,l.next=o),e.pending=o}function y0(e,o,l){if((l&4194240)!==0){var d=o.lanes;d&=e.pendingLanes,l|=d,o.lanes=l,Yl(e,l)}}var qi={readContext:wr,useCallback:Gt,useContext:Gt,useEffect:Gt,useImperativeHandle:Gt,useInsertionEffect:Gt,useLayoutEffect:Gt,useMemo:Gt,useReducer:Gt,useRef:Gt,useState:Gt,useDebugValue:Gt,useDeferredValue:Gt,useTransition:Gt,useMutableSource:Gt,useSyncExternalStore:Gt,useId:Gt,unstable_isNewReconciler:!1},fv={readContext:wr,useCallback:function(e,o){return zr().memoizedState=[e,o===void 0?null:o],e},useContext:wr,useEffect:l0,useImperativeHandle:function(e,o,l){return l=l!=null?l.concat([e]):null,Wi(4194308,4,d0.bind(null,o,e),l)},useLayoutEffect:function(e,o){return Wi(4194308,4,e,o)},useInsertionEffect:function(e,o){return Wi(4,2,e,o)},useMemo:function(e,o){var l=zr();return o=o===void 0?null:o,e=e(),l.memoizedState=[e,o],e},useReducer:function(e,o,l){var d=zr();return o=l!==void 0?l(o):o,d.memoizedState=d.baseState=o,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:o},d.queue=e,e=e.dispatch=uv.bind(null,Et,e),[d.memoizedState,e]},useRef:function(e){var o=zr();return e={current:e},o.memoizedState=e},useState:i0,useDebugValue:Kc,useDeferredValue:function(e){return zr().memoizedState=e},useTransition:function(){var e=i0(!1),o=e[0];return e=cv.bind(null,e[1]),zr().memoizedState=e,[o,e]},useMutableSource:function(){},useSyncExternalStore:function(e,o,l){var d=Et,m=zr();if(yt){if(l===void 0)throw Error(n(407));l=l()}else{if(l=o(),Ot===null)throw Error(n(349));(fo&30)!==0||t0(d,o,l)}m.memoizedState=l;var v={value:l,getSnapshot:o};return m.queue=v,l0(n0.bind(null,d,v,e),[e]),d.flags|=2048,La(9,r0.bind(null,d,v,l,o),void 0,null),l},useId:function(){var e=zr(),o=Ot.identifierPrefix;if(yt){var l=rn,d=tn;l=(d&~(1<<32-ut(d)-1)).toString(32)+l,o=":"+o+"R"+l,l=Ta++,0<l&&(o+="H"+l.toString(32)),o+=":"}else l=lv++,o=":"+o+"r"+l.toString(32)+":";return e.memoizedState=o},unstable_isNewReconciler:!1},pv={readContext:wr,useCallback:p0,useContext:wr,useEffect:qc,useImperativeHandle:f0,useInsertionEffect:c0,useLayoutEffect:u0,useMemo:h0,useReducer:Wc,useRef:s0,useState:function(){return Wc(_a)},useDebugValue:Kc,useDeferredValue:function(e){var o=Er();return m0(o,Nt.memoizedState,e)},useTransition:function(){var e=Wc(_a)[0],o=Er().memoizedState;return[e,o]},useMutableSource:Zf,useSyncExternalStore:e0,useId:g0,unstable_isNewReconciler:!1},hv={readContext:wr,useCallback:p0,useContext:wr,useEffect:qc,useImperativeHandle:f0,useInsertionEffect:c0,useLayoutEffect:u0,useMemo:h0,useReducer:Gc,useRef:s0,useState:function(){return Gc(_a)},useDebugValue:Kc,useDeferredValue:function(e){var o=Er();return Nt===null?o.memoizedState=e:m0(o,Nt.memoizedState,e)},useTransition:function(){var e=Gc(_a)[0],o=Er().memoizedState;return[e,o]},useMutableSource:Zf,useSyncExternalStore:e0,useId:g0,unstable_isNewReconciler:!1};function _r(e,o){if(e&&e.defaultProps){o=ee({},o),e=e.defaultProps;for(var l in e)o[l]===void 0&&(o[l]=e[l]);return o}return o}function Xc(e,o,l,d){o=e.memoizedState,l=l(d,o),l=l==null?o:ee({},o,l),e.memoizedState=l,e.lanes===0&&(e.updateQueue.baseState=l)}var Ki={isMounted:function(e){return(e=e._reactInternals)?Ae(e)===e:!1},enqueueSetState:function(e,o,l){e=e._reactInternals;var d=Zt(),m=Mn(e),v=on(d,m);v.payload=o,l!=null&&(v.callback=l),o=In(e,v,m),o!==null&&(Nr(o,e,m,d),$i(o,e,m))},enqueueReplaceState:function(e,o,l){e=e._reactInternals;var d=Zt(),m=Mn(e),v=on(d,m);v.tag=1,v.payload=o,l!=null&&(v.callback=l),o=In(e,v,m),o!==null&&(Nr(o,e,m,d),$i(o,e,m))},enqueueForceUpdate:function(e,o){e=e._reactInternals;var l=Zt(),d=Mn(e),m=on(l,d);m.tag=2,o!=null&&(m.callback=o),o=In(e,m,d),o!==null&&(Nr(o,e,d,l),$i(o,e,d))}};function w0(e,o,l,d,m,v,R){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(d,v,R):o.prototype&&o.prototype.isPureReactComponent?!va(l,d)||!va(m,v):!0}function E0(e,o,l){var d=!1,m=Ln,v=o.contextType;return typeof v=="object"&&v!==null?v=wr(v):(m=nr(o)?io:Wt.current,d=o.contextTypes,v=(d=d!=null)?Mo(e,m):Ln),o=new o(l,v),e.memoizedState=o.state!==null&&o.state!==void 0?o.state:null,o.updater=Ki,e.stateNode=o,o._reactInternals=e,d&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=m,e.__reactInternalMemoizedMaskedChildContext=v),o}function C0(e,o,l,d){e=o.state,typeof o.componentWillReceiveProps=="function"&&o.componentWillReceiveProps(l,d),typeof o.UNSAFE_componentWillReceiveProps=="function"&&o.UNSAFE_componentWillReceiveProps(l,d),o.state!==e&&Ki.enqueueReplaceState(o,o.state,null)}function Yc(e,o,l,d){var m=e.stateNode;m.props=l,m.state=e.memoizedState,m.refs={},Bc(e);var v=o.contextType;typeof v=="object"&&v!==null?m.context=wr(v):(v=nr(o)?io:Wt.current,m.context=Mo(e,v)),m.state=e.memoizedState,v=o.getDerivedStateFromProps,typeof v=="function"&&(Xc(e,o,v,l),m.state=e.memoizedState),typeof o.getDerivedStateFromProps=="function"||typeof m.getSnapshotBeforeUpdate=="function"||typeof m.UNSAFE_componentWillMount!="function"&&typeof m.componentWillMount!="function"||(o=m.state,typeof m.componentWillMount=="function"&&m.componentWillMount(),typeof m.UNSAFE_componentWillMount=="function"&&m.UNSAFE_componentWillMount(),o!==m.state&&Ki.enqueueReplaceState(m,m.state,null),Ui(e,l,m,d),m.state=e.memoizedState),typeof m.componentDidMount=="function"&&(e.flags|=4194308)}function qo(e,o){try{var l="",d=o;do l+=ue(d),d=d.return;while(d);var m=l}catch(v){m=`
Error generating stack: `+v.message+`
`+v.stack}return{value:e,source:o,stack:m,digest:null}}function Qc(e,o,l){return{value:e,source:null,stack:l??null,digest:o??null}}function Jc(e,o){try{console.error(o.value)}catch(l){setTimeout(function(){throw l})}}var mv=typeof WeakMap=="function"?WeakMap:Map;function b0(e,o,l){l=on(-1,l),l.tag=3,l.payload={element:null};var d=o.value;return l.callback=function(){ts||(ts=!0,pu=d),Jc(e,o)},l}function S0(e,o,l){l=on(-1,l),l.tag=3;var d=e.type.getDerivedStateFromError;if(typeof d=="function"){var m=o.value;l.payload=function(){return d(m)},l.callback=function(){Jc(e,o)}}var v=e.stateNode;return v!==null&&typeof v.componentDidCatch=="function"&&(l.callback=function(){Jc(e,o),typeof d!="function"&&(jn===null?jn=new Set([this]):jn.add(this));var R=o.stack;this.componentDidCatch(o.value,{componentStack:R!==null?R:""})}),l}function R0(e,o,l){var d=e.pingCache;if(d===null){d=e.pingCache=new mv;var m=new Set;d.set(o,m)}else m=d.get(o),m===void 0&&(m=new Set,d.set(o,m));m.has(l)||(m.add(l),e=Dv.bind(null,e,o,l),o.then(e,e))}function A0(e){do{var o;if((o=e.tag===13)&&(o=e.memoizedState,o=o!==null?o.dehydrated!==null:!0),o)return e;e=e.return}while(e!==null);return null}function P0(e,o,l,d,m){return(e.mode&1)===0?(e===o?e.flags|=65536:(e.flags|=128,l.flags|=131072,l.flags&=-52805,l.tag===1&&(l.alternate===null?l.tag=17:(o=on(-1,1),o.tag=2,In(l,o,1))),l.lanes|=1),e):(e.flags|=65536,e.lanes=m,e)}var gv=T.ReactCurrentOwner,or=!1;function Jt(e,o,l,d){o.child=e===null?qf(o,null,l,d):Ho(o,e.child,l,d)}function k0(e,o,l,d,m){l=l.render;var v=o.ref;return Wo(o,m),d=Hc(e,o,l,d,v,m),l=Vc(),e!==null&&!or?(o.updateQueue=e.updateQueue,o.flags&=-2053,e.lanes&=~m,an(e,o,m)):(yt&&l&&Ac(o),o.flags|=1,Jt(e,o,d,m),o.child)}function D0(e,o,l,d,m){if(e===null){var v=l.type;return typeof v=="function"&&!wu(v)&&v.defaultProps===void 0&&l.compare===null&&l.defaultProps===void 0?(o.tag=15,o.type=v,T0(e,o,v,d,m)):(e=ss(l.type,null,d,o,o.mode,m),e.ref=o.ref,e.return=o,o.child=e)}if(v=e.child,(e.lanes&m)===0){var R=v.memoizedProps;if(l=l.compare,l=l!==null?l:va,l(R,d)&&e.ref===o.ref)return an(e,o,m)}return o.flags|=1,e=Un(v,d),e.ref=o.ref,e.return=o,o.child=e}function T0(e,o,l,d,m){if(e!==null){var v=e.memoizedProps;if(va(v,d)&&e.ref===o.ref)if(or=!1,o.pendingProps=d=v,(e.lanes&m)!==0)(e.flags&131072)!==0&&(or=!0);else return o.lanes=e.lanes,an(e,o,m)}return Zc(e,o,l,d,m)}function _0(e,o,l){var d=o.pendingProps,m=d.children,v=e!==null?e.memoizedState:null;if(d.mode==="hidden")if((o.mode&1)===0)o.memoizedState={baseLanes:0,cachePool:null,transitions:null},pt(Xo,mr),mr|=l;else{if((l&1073741824)===0)return e=v!==null?v.baseLanes|l:l,o.lanes=o.childLanes=1073741824,o.memoizedState={baseLanes:e,cachePool:null,transitions:null},o.updateQueue=null,pt(Xo,mr),mr|=e,null;o.memoizedState={baseLanes:0,cachePool:null,transitions:null},d=v!==null?v.baseLanes:l,pt(Xo,mr),mr|=d}else v!==null?(d=v.baseLanes|l,o.memoizedState=null):d=l,pt(Xo,mr),mr|=d;return Jt(e,o,m,l),o.child}function L0(e,o){var l=o.ref;(e===null&&l!==null||e!==null&&e.ref!==l)&&(o.flags|=512,o.flags|=2097152)}function Zc(e,o,l,d,m){var v=nr(l)?io:Wt.current;return v=Mo(o,v),Wo(o,m),l=Hc(e,o,l,d,v,m),d=Vc(),e!==null&&!or?(o.updateQueue=e.updateQueue,o.flags&=-2053,e.lanes&=~m,an(e,o,m)):(yt&&d&&Ac(o),o.flags|=1,Jt(e,o,l,m),o.child)}function F0(e,o,l,d,m){if(nr(l)){var v=!0;Li(o)}else v=!1;if(Wo(o,m),o.stateNode===null)Yi(e,o),E0(o,l,d),Yc(o,l,d,m),d=!0;else if(e===null){var R=o.stateNode,j=o.memoizedProps;R.props=j;var U=R.context,J=l.contextType;typeof J=="object"&&J!==null?J=wr(J):(J=nr(l)?io:Wt.current,J=Mo(o,J));var le=l.getDerivedStateFromProps,ce=typeof le=="function"||typeof R.getSnapshotBeforeUpdate=="function";ce||typeof R.UNSAFE_componentWillReceiveProps!="function"&&typeof R.componentWillReceiveProps!="function"||(j!==d||U!==J)&&C0(o,R,d,J),Nn=!1;var ie=o.memoizedState;R.state=ie,Ui(o,d,R,m),U=o.memoizedState,j!==d||ie!==U||rr.current||Nn?(typeof le=="function"&&(Xc(o,l,le,d),U=o.memoizedState),(j=Nn||w0(o,l,j,d,ie,U,J))?(ce||typeof R.UNSAFE_componentWillMount!="function"&&typeof R.componentWillMount!="function"||(typeof R.componentWillMount=="function"&&R.componentWillMount(),typeof R.UNSAFE_componentWillMount=="function"&&R.UNSAFE_componentWillMount()),typeof R.componentDidMount=="function"&&(o.flags|=4194308)):(typeof R.componentDidMount=="function"&&(o.flags|=4194308),o.memoizedProps=d,o.memoizedState=U),R.props=d,R.state=U,R.context=J,d=j):(typeof R.componentDidMount=="function"&&(o.flags|=4194308),d=!1)}else{R=o.stateNode,Xf(e,o),j=o.memoizedProps,J=o.type===o.elementType?j:_r(o.type,j),R.props=J,ce=o.pendingProps,ie=R.context,U=l.contextType,typeof U=="object"&&U!==null?U=wr(U):(U=nr(l)?io:Wt.current,U=Mo(o,U));var we=l.getDerivedStateFromProps;(le=typeof we=="function"||typeof R.getSnapshotBeforeUpdate=="function")||typeof R.UNSAFE_componentWillReceiveProps!="function"&&typeof R.componentWillReceiveProps!="function"||(j!==ce||ie!==U)&&C0(o,R,d,U),Nn=!1,ie=o.memoizedState,R.state=ie,Ui(o,d,R,m);var Re=o.memoizedState;j!==ce||ie!==Re||rr.current||Nn?(typeof we=="function"&&(Xc(o,l,we,d),Re=o.memoizedState),(J=Nn||w0(o,l,J,d,ie,Re,U)||!1)?(le||typeof R.UNSAFE_componentWillUpdate!="function"&&typeof R.componentWillUpdate!="function"||(typeof R.componentWillUpdate=="function"&&R.componentWillUpdate(d,Re,U),typeof R.UNSAFE_componentWillUpdate=="function"&&R.UNSAFE_componentWillUpdate(d,Re,U)),typeof R.componentDidUpdate=="function"&&(o.flags|=4),typeof R.getSnapshotBeforeUpdate=="function"&&(o.flags|=1024)):(typeof R.componentDidUpdate!="function"||j===e.memoizedProps&&ie===e.memoizedState||(o.flags|=4),typeof R.getSnapshotBeforeUpdate!="function"||j===e.memoizedProps&&ie===e.memoizedState||(o.flags|=1024),o.memoizedProps=d,o.memoizedState=Re),R.props=d,R.state=Re,R.context=U,d=J):(typeof R.componentDidUpdate!="function"||j===e.memoizedProps&&ie===e.memoizedState||(o.flags|=4),typeof R.getSnapshotBeforeUpdate!="function"||j===e.memoizedProps&&ie===e.memoizedState||(o.flags|=1024),d=!1)}return eu(e,o,l,d,v,m)}function eu(e,o,l,d,m,v){L0(e,o);var R=(o.flags&128)!==0;if(!d&&!R)return m&&Of(o,l,!1),an(e,o,v);d=o.stateNode,gv.current=o;var j=R&&typeof l.getDerivedStateFromError!="function"?null:d.render();return o.flags|=1,e!==null&&R?(o.child=Ho(o,e.child,null,v),o.child=Ho(o,null,j,v)):Jt(e,o,j,v),o.memoizedState=d.state,m&&Of(o,l,!0),o.child}function N0(e){var o=e.stateNode;o.pendingContext?Bf(e,o.pendingContext,o.pendingContext!==o.context):o.context&&Bf(e,o.context,!1),jc(e,o.containerInfo)}function I0(e,o,l,d,m){return zo(),Tc(m),o.flags|=256,Jt(e,o,l,d),o.child}var tu={dehydrated:null,treeContext:null,retryLane:0};function ru(e){return{baseLanes:e,cachePool:null,transitions:null}}function B0(e,o,l){var d=o.pendingProps,m=wt.current,v=!1,R=(o.flags&128)!==0,j;if((j=R)||(j=e!==null&&e.memoizedState===null?!1:(m&2)!==0),j?(v=!0,o.flags&=-129):(e===null||e.memoizedState!==null)&&(m|=1),pt(wt,m&1),e===null)return Dc(o),e=o.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?((o.mode&1)===0?o.lanes=1:e.data==="$!"?o.lanes=8:o.lanes=1073741824,null):(R=d.children,e=d.fallback,v?(d=o.mode,v=o.child,R={mode:"hidden",children:R},(d&1)===0&&v!==null?(v.childLanes=0,v.pendingProps=R):v=ls(R,d,0,null),e=xo(e,d,l,null),v.return=o,e.return=o,v.sibling=e,o.child=v,o.child.memoizedState=ru(l),o.memoizedState=tu,e):nu(o,R));if(m=e.memoizedState,m!==null&&(j=m.dehydrated,j!==null))return xv(e,o,R,d,j,m,l);if(v){v=d.fallback,R=o.mode,m=e.child,j=m.sibling;var U={mode:"hidden",children:d.children};return(R&1)===0&&o.child!==m?(d=o.child,d.childLanes=0,d.pendingProps=U,o.deletions=null):(d=Un(m,U),d.subtreeFlags=m.subtreeFlags&14680064),j!==null?v=Un(j,v):(v=xo(v,R,l,null),v.flags|=2),v.return=o,d.return=o,d.sibling=v,o.child=d,d=v,v=o.child,R=e.child.memoizedState,R=R===null?ru(l):{baseLanes:R.baseLanes|l,cachePool:null,transitions:R.transitions},v.memoizedState=R,v.childLanes=e.childLanes&~l,o.memoizedState=tu,d}return v=e.child,e=v.sibling,d=Un(v,{mode:"visible",children:d.children}),(o.mode&1)===0&&(d.lanes=l),d.return=o,d.sibling=null,e!==null&&(l=o.deletions,l===null?(o.deletions=[e],o.flags|=16):l.push(e)),o.child=d,o.memoizedState=null,d}function nu(e,o){return o=ls({mode:"visible",children:o},e.mode,0,null),o.return=e,e.child=o}function Xi(e,o,l,d){return d!==null&&Tc(d),Ho(o,e.child,null,l),e=nu(o,o.pendingProps.children),e.flags|=2,o.memoizedState=null,e}function xv(e,o,l,d,m,v,R){if(l)return o.flags&256?(o.flags&=-257,d=Qc(Error(n(422))),Xi(e,o,R,d)):o.memoizedState!==null?(o.child=e.child,o.flags|=128,null):(v=d.fallback,m=o.mode,d=ls({mode:"visible",children:d.children},m,0,null),v=xo(v,m,R,null),v.flags|=2,d.return=o,v.return=o,d.sibling=v,o.child=d,(o.mode&1)!==0&&Ho(o,e.child,null,R),o.child.memoizedState=ru(R),o.memoizedState=tu,v);if((o.mode&1)===0)return Xi(e,o,R,null);if(m.data==="$!"){if(d=m.nextSibling&&m.nextSibling.dataset,d)var j=d.dgst;return d=j,v=Error(n(419)),d=Qc(v,d,void 0),Xi(e,o,R,d)}if(j=(R&e.childLanes)!==0,or||j){if(d=Ot,d!==null){switch(R&-R){case 4:m=2;break;case 16:m=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:m=32;break;case 536870912:m=268435456;break;default:m=0}m=(m&(d.suspendedLanes|R))!==0?0:m,m!==0&&m!==v.retryLane&&(v.retryLane=m,nn(e,m),Nr(d,e,m,-1))}return yu(),d=Qc(Error(n(421))),Xi(e,o,R,d)}return m.data==="$?"?(o.flags|=128,o.child=e.child,o=Tv.bind(null,e),m._reactRetry=o,null):(e=v.treeContext,hr=Tn(m.nextSibling),pr=o,yt=!0,Tr=null,e!==null&&(vr[yr++]=tn,vr[yr++]=rn,vr[yr++]=so,tn=e.id,rn=e.overflow,so=o),o=nu(o,d.children),o.flags|=4096,o)}function j0(e,o,l){e.lanes|=o;var d=e.alternate;d!==null&&(d.lanes|=o),Nc(e.return,o,l)}function ou(e,o,l,d,m){var v=e.memoizedState;v===null?e.memoizedState={isBackwards:o,rendering:null,renderingStartTime:0,last:d,tail:l,tailMode:m}:(v.isBackwards=o,v.rendering=null,v.renderingStartTime=0,v.last=d,v.tail=l,v.tailMode=m)}function O0(e,o,l){var d=o.pendingProps,m=d.revealOrder,v=d.tail;if(Jt(e,o,d.children,l),d=wt.current,(d&2)!==0)d=d&1|2,o.flags|=128;else{if(e!==null&&(e.flags&128)!==0)e:for(e=o.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&j0(e,l,o);else if(e.tag===19)j0(e,l,o);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===o)break e;for(;e.sibling===null;){if(e.return===null||e.return===o)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}d&=1}if(pt(wt,d),(o.mode&1)===0)o.memoizedState=null;else switch(m){case"forwards":for(l=o.child,m=null;l!==null;)e=l.alternate,e!==null&&zi(e)===null&&(m=l),l=l.sibling;l=m,l===null?(m=o.child,o.child=null):(m=l.sibling,l.sibling=null),ou(o,!1,m,l,v);break;case"backwards":for(l=null,m=o.child,o.child=null;m!==null;){if(e=m.alternate,e!==null&&zi(e)===null){o.child=m;break}e=m.sibling,m.sibling=l,l=m,m=e}ou(o,!0,l,null,v);break;case"together":ou(o,!1,null,null,void 0);break;default:o.memoizedState=null}return o.child}function Yi(e,o){(o.mode&1)===0&&e!==null&&(e.alternate=null,o.alternate=null,o.flags|=2)}function an(e,o,l){if(e!==null&&(o.dependencies=e.dependencies),po|=o.lanes,(l&o.childLanes)===0)return null;if(e!==null&&o.child!==e.child)throw Error(n(153));if(o.child!==null){for(e=o.child,l=Un(e,e.pendingProps),o.child=l,l.return=o;e.sibling!==null;)e=e.sibling,l=l.sibling=Un(e,e.pendingProps),l.return=o;l.sibling=null}return o.child}function vv(e,o,l){switch(o.tag){case 3:N0(o),zo();break;case 5:Jf(o);break;case 1:nr(o.type)&&Li(o);break;case 4:jc(o,o.stateNode.containerInfo);break;case 10:var d=o.type._context,m=o.memoizedProps.value;pt(Oi,d._currentValue),d._currentValue=m;break;case 13:if(d=o.memoizedState,d!==null)return d.dehydrated!==null?(pt(wt,wt.current&1),o.flags|=128,null):(l&o.child.childLanes)!==0?B0(e,o,l):(pt(wt,wt.current&1),e=an(e,o,l),e!==null?e.sibling:null);pt(wt,wt.current&1);break;case 19:if(d=(l&o.childLanes)!==0,(e.flags&128)!==0){if(d)return O0(e,o,l);o.flags|=128}if(m=o.memoizedState,m!==null&&(m.rendering=null,m.tail=null,m.lastEffect=null),pt(wt,wt.current),d)break;return null;case 22:case 23:return o.lanes=0,_0(e,o,l)}return an(e,o,l)}var M0,au,$0,U0;M0=function(e,o){for(var l=o.child;l!==null;){if(l.tag===5||l.tag===6)e.appendChild(l.stateNode);else if(l.tag!==4&&l.child!==null){l.child.return=l,l=l.child;continue}if(l===o)break;for(;l.sibling===null;){if(l.return===null||l.return===o)return;l=l.return}l.sibling.return=l.return,l=l.sibling}},au=function(){},$0=function(e,o,l,d){var m=e.memoizedProps;if(m!==d){e=o.stateNode,uo(Ur.current);var v=null;switch(l){case"input":m=ge(e,m),d=ge(e,d),v=[];break;case"select":m=ee({},m,{value:void 0}),d=ee({},d,{value:void 0}),v=[];break;case"textarea":m=Fe(e,m),d=Fe(e,d),v=[];break;default:typeof m.onClick!="function"&&typeof d.onClick=="function"&&(e.onclick=Di)}mn(l,d);var R;l=null;for(J in m)if(!d.hasOwnProperty(J)&&m.hasOwnProperty(J)&&m[J]!=null)if(J==="style"){var j=m[J];for(R in j)j.hasOwnProperty(R)&&(l||(l={}),l[R]="")}else J!=="dangerouslySetInnerHTML"&&J!=="children"&&J!=="suppressContentEditableWarning"&&J!=="suppressHydrationWarning"&&J!=="autoFocus"&&(i.hasOwnProperty(J)?v||(v=[]):(v=v||[]).push(J,null));for(J in d){var U=d[J];if(j=m!=null?m[J]:void 0,d.hasOwnProperty(J)&&U!==j&&(U!=null||j!=null))if(J==="style")if(j){for(R in j)!j.hasOwnProperty(R)||U&&U.hasOwnProperty(R)||(l||(l={}),l[R]="");for(R in U)U.hasOwnProperty(R)&&j[R]!==U[R]&&(l||(l={}),l[R]=U[R])}else l||(v||(v=[]),v.push(J,l)),l=U;else J==="dangerouslySetInnerHTML"?(U=U?U.__html:void 0,j=j?j.__html:void 0,U!=null&&j!==U&&(v=v||[]).push(J,U)):J==="children"?typeof U!="string"&&typeof U!="number"||(v=v||[]).push(J,""+U):J!=="suppressContentEditableWarning"&&J!=="suppressHydrationWarning"&&(i.hasOwnProperty(J)?(U!=null&&J==="onScroll"&&gt("scroll",e),v||j===U||(v=[])):(v=v||[]).push(J,U))}l&&(v=v||[]).push("style",l);var J=v;(o.updateQueue=J)&&(o.flags|=4)}},U0=function(e,o,l,d){l!==d&&(o.flags|=4)};function Fa(e,o){if(!yt)switch(e.tailMode){case"hidden":o=e.tail;for(var l=null;o!==null;)o.alternate!==null&&(l=o),o=o.sibling;l===null?e.tail=null:l.sibling=null;break;case"collapsed":l=e.tail;for(var d=null;l!==null;)l.alternate!==null&&(d=l),l=l.sibling;d===null?o||e.tail===null?e.tail=null:e.tail.sibling=null:d.sibling=null}}function qt(e){var o=e.alternate!==null&&e.alternate.child===e.child,l=0,d=0;if(o)for(var m=e.child;m!==null;)l|=m.lanes|m.childLanes,d|=m.subtreeFlags&14680064,d|=m.flags&14680064,m.return=e,m=m.sibling;else for(m=e.child;m!==null;)l|=m.lanes|m.childLanes,d|=m.subtreeFlags,d|=m.flags,m.return=e,m=m.sibling;return e.subtreeFlags|=d,e.childLanes=l,o}function yv(e,o,l){var d=o.pendingProps;switch(Pc(o),o.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return qt(o),null;case 1:return nr(o.type)&&_i(),qt(o),null;case 3:return d=o.stateNode,Go(),xt(rr),xt(Wt),$c(),d.pendingContext&&(d.context=d.pendingContext,d.pendingContext=null),(e===null||e.child===null)&&(Bi(o)?o.flags|=4:e===null||e.memoizedState.isDehydrated&&(o.flags&256)===0||(o.flags|=1024,Tr!==null&&(gu(Tr),Tr=null))),au(e,o),qt(o),null;case 5:Oc(o);var m=uo(ka.current);if(l=o.type,e!==null&&o.stateNode!=null)$0(e,o,l,d,m),e.ref!==o.ref&&(o.flags|=512,o.flags|=2097152);else{if(!d){if(o.stateNode===null)throw Error(n(166));return qt(o),null}if(e=uo(Ur.current),Bi(o)){d=o.stateNode,l=o.type;var v=o.memoizedProps;switch(d[$r]=o,d[ba]=v,e=(o.mode&1)!==0,l){case"dialog":gt("cancel",d),gt("close",d);break;case"iframe":case"object":case"embed":gt("load",d);break;case"video":case"audio":for(m=0;m<wa.length;m++)gt(wa[m],d);break;case"source":gt("error",d);break;case"img":case"image":case"link":gt("error",d),gt("load",d);break;case"details":gt("toggle",d);break;case"input":je(d,v),gt("invalid",d);break;case"select":d._wrapperState={wasMultiple:!!v.multiple},gt("invalid",d);break;case"textarea":He(d,v),gt("invalid",d)}mn(l,v),m=null;for(var R in v)if(v.hasOwnProperty(R)){var j=v[R];R==="children"?typeof j=="string"?d.textContent!==j&&(v.suppressHydrationWarning!==!0&&ki(d.textContent,j,e),m=["children",j]):typeof j=="number"&&d.textContent!==""+j&&(v.suppressHydrationWarning!==!0&&ki(d.textContent,j,e),m=["children",""+j]):i.hasOwnProperty(R)&&j!=null&&R==="onScroll"&&gt("scroll",d)}switch(l){case"input":Vt(d),Dt(d,v,!0);break;case"textarea":Vt(d),ct(d);break;case"select":case"option":break;default:typeof v.onClick=="function"&&(d.onclick=Di)}d=m,o.updateQueue=d,d!==null&&(o.flags|=4)}else{R=m.nodeType===9?m:m.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=rt(l)),e==="http://www.w3.org/1999/xhtml"?l==="script"?(e=R.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof d.is=="string"?e=R.createElement(l,{is:d.is}):(e=R.createElement(l),l==="select"&&(R=e,d.multiple?R.multiple=!0:d.size&&(R.size=d.size))):e=R.createElementNS(e,l),e[$r]=o,e[ba]=d,M0(e,o,!1,!1),o.stateNode=e;e:{switch(R=gn(l,d),l){case"dialog":gt("cancel",e),gt("close",e),m=d;break;case"iframe":case"object":case"embed":gt("load",e),m=d;break;case"video":case"audio":for(m=0;m<wa.length;m++)gt(wa[m],e);m=d;break;case"source":gt("error",e),m=d;break;case"img":case"image":case"link":gt("error",e),gt("load",e),m=d;break;case"details":gt("toggle",e),m=d;break;case"input":je(e,d),m=ge(e,d),gt("invalid",e);break;case"option":m=d;break;case"select":e._wrapperState={wasMultiple:!!d.multiple},m=ee({},d,{value:void 0}),gt("invalid",e);break;case"textarea":He(e,d),m=Fe(e,d),gt("invalid",e);break;default:m=d}mn(l,m),j=m;for(v in j)if(j.hasOwnProperty(v)){var U=j[v];v==="style"?Yt(e,U):v==="dangerouslySetInnerHTML"?(U=U?U.__html:void 0,U!=null&&mt(e,U)):v==="children"?typeof U=="string"?(l!=="textarea"||U!=="")&&Tt(e,U):typeof U=="number"&&Tt(e,""+U):v!=="suppressContentEditableWarning"&&v!=="suppressHydrationWarning"&&v!=="autoFocus"&&(i.hasOwnProperty(v)?U!=null&&v==="onScroll"&&gt("scroll",e):U!=null&&A(e,v,U,R))}switch(l){case"input":Vt(e),Dt(e,d,!1);break;case"textarea":Vt(e),ct(e);break;case"option":d.value!=null&&e.setAttribute("value",""+he(d.value));break;case"select":e.multiple=!!d.multiple,v=d.value,v!=null?ye(e,!!d.multiple,v,!1):d.defaultValue!=null&&ye(e,!!d.multiple,d.defaultValue,!0);break;default:typeof m.onClick=="function"&&(e.onclick=Di)}switch(l){case"button":case"input":case"select":case"textarea":d=!!d.autoFocus;break e;case"img":d=!0;break e;default:d=!1}}d&&(o.flags|=4)}o.ref!==null&&(o.flags|=512,o.flags|=2097152)}return qt(o),null;case 6:if(e&&o.stateNode!=null)U0(e,o,e.memoizedProps,d);else{if(typeof d!="string"&&o.stateNode===null)throw Error(n(166));if(l=uo(ka.current),uo(Ur.current),Bi(o)){if(d=o.stateNode,l=o.memoizedProps,d[$r]=o,(v=d.nodeValue!==l)&&(e=pr,e!==null))switch(e.tag){case 3:ki(d.nodeValue,l,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&ki(d.nodeValue,l,(e.mode&1)!==0)}v&&(o.flags|=4)}else d=(l.nodeType===9?l:l.ownerDocument).createTextNode(d),d[$r]=o,o.stateNode=d}return qt(o),null;case 13:if(xt(wt),d=o.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(yt&&hr!==null&&(o.mode&1)!==0&&(o.flags&128)===0)Vf(),zo(),o.flags|=98560,v=!1;else if(v=Bi(o),d!==null&&d.dehydrated!==null){if(e===null){if(!v)throw Error(n(318));if(v=o.memoizedState,v=v!==null?v.dehydrated:null,!v)throw Error(n(317));v[$r]=o}else zo(),(o.flags&128)===0&&(o.memoizedState=null),o.flags|=4;qt(o),v=!1}else Tr!==null&&(gu(Tr),Tr=null),v=!0;if(!v)return o.flags&65536?o:null}return(o.flags&128)!==0?(o.lanes=l,o):(d=d!==null,d!==(e!==null&&e.memoizedState!==null)&&d&&(o.child.flags|=8192,(o.mode&1)!==0&&(e===null||(wt.current&1)!==0?It===0&&(It=3):yu())),o.updateQueue!==null&&(o.flags|=4),qt(o),null);case 4:return Go(),au(e,o),e===null&&Ea(o.stateNode.containerInfo),qt(o),null;case 10:return Fc(o.type._context),qt(o),null;case 17:return nr(o.type)&&_i(),qt(o),null;case 19:if(xt(wt),v=o.memoizedState,v===null)return qt(o),null;if(d=(o.flags&128)!==0,R=v.rendering,R===null)if(d)Fa(v,!1);else{if(It!==0||e!==null&&(e.flags&128)!==0)for(e=o.child;e!==null;){if(R=zi(e),R!==null){for(o.flags|=128,Fa(v,!1),d=R.updateQueue,d!==null&&(o.updateQueue=d,o.flags|=4),o.subtreeFlags=0,d=l,l=o.child;l!==null;)v=l,e=d,v.flags&=14680066,R=v.alternate,R===null?(v.childLanes=0,v.lanes=e,v.child=null,v.subtreeFlags=0,v.memoizedProps=null,v.memoizedState=null,v.updateQueue=null,v.dependencies=null,v.stateNode=null):(v.childLanes=R.childLanes,v.lanes=R.lanes,v.child=R.child,v.subtreeFlags=0,v.deletions=null,v.memoizedProps=R.memoizedProps,v.memoizedState=R.memoizedState,v.updateQueue=R.updateQueue,v.type=R.type,e=R.dependencies,v.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),l=l.sibling;return pt(wt,wt.current&1|2),o.child}e=e.sibling}v.tail!==null&&at()>Yo&&(o.flags|=128,d=!0,Fa(v,!1),o.lanes=4194304)}else{if(!d)if(e=zi(R),e!==null){if(o.flags|=128,d=!0,l=e.updateQueue,l!==null&&(o.updateQueue=l,o.flags|=4),Fa(v,!0),v.tail===null&&v.tailMode==="hidden"&&!R.alternate&&!yt)return qt(o),null}else 2*at()-v.renderingStartTime>Yo&&l!==1073741824&&(o.flags|=128,d=!0,Fa(v,!1),o.lanes=4194304);v.isBackwards?(R.sibling=o.child,o.child=R):(l=v.last,l!==null?l.sibling=R:o.child=R,v.last=R)}return v.tail!==null?(o=v.tail,v.rendering=o,v.tail=o.sibling,v.renderingStartTime=at(),o.sibling=null,l=wt.current,pt(wt,d?l&1|2:l&1),o):(qt(o),null);case 22:case 23:return vu(),d=o.memoizedState!==null,e!==null&&e.memoizedState!==null!==d&&(o.flags|=8192),d&&(o.mode&1)!==0?(mr&1073741824)!==0&&(qt(o),o.subtreeFlags&6&&(o.flags|=8192)):qt(o),null;case 24:return null;case 25:return null}throw Error(n(156,o.tag))}function wv(e,o){switch(Pc(o),o.tag){case 1:return nr(o.type)&&_i(),e=o.flags,e&65536?(o.flags=e&-65537|128,o):null;case 3:return Go(),xt(rr),xt(Wt),$c(),e=o.flags,(e&65536)!==0&&(e&128)===0?(o.flags=e&-65537|128,o):null;case 5:return Oc(o),null;case 13:if(xt(wt),e=o.memoizedState,e!==null&&e.dehydrated!==null){if(o.alternate===null)throw Error(n(340));zo()}return e=o.flags,e&65536?(o.flags=e&-65537|128,o):null;case 19:return xt(wt),null;case 4:return Go(),null;case 10:return Fc(o.type._context),null;case 22:case 23:return vu(),null;case 24:return null;default:return null}}var Qi=!1,Kt=!1,Ev=typeof WeakSet=="function"?WeakSet:Set,be=null;function Ko(e,o){var l=e.ref;if(l!==null)if(typeof l=="function")try{l(null)}catch(d){St(e,o,d)}else l.current=null}function iu(e,o,l){try{l()}catch(d){St(e,o,d)}}var z0=!1;function Cv(e,o){if(vc=xi,e=wf(),uc(e)){if("selectionStart"in e)var l={start:e.selectionStart,end:e.selectionEnd};else e:{l=(l=e.ownerDocument)&&l.defaultView||window;var d=l.getSelection&&l.getSelection();if(d&&d.rangeCount!==0){l=d.anchorNode;var m=d.anchorOffset,v=d.focusNode;d=d.focusOffset;try{l.nodeType,v.nodeType}catch{l=null;break e}var R=0,j=-1,U=-1,J=0,le=0,ce=e,ie=null;t:for(;;){for(var we;ce!==l||m!==0&&ce.nodeType!==3||(j=R+m),ce!==v||d!==0&&ce.nodeType!==3||(U=R+d),ce.nodeType===3&&(R+=ce.nodeValue.length),(we=ce.firstChild)!==null;)ie=ce,ce=we;for(;;){if(ce===e)break t;if(ie===l&&++J===m&&(j=R),ie===v&&++le===d&&(U=R),(we=ce.nextSibling)!==null)break;ce=ie,ie=ce.parentNode}ce=we}l=j===-1||U===-1?null:{start:j,end:U}}else l=null}l=l||{start:0,end:0}}else l=null;for(yc={focusedElem:e,selectionRange:l},xi=!1,be=o;be!==null;)if(o=be,e=o.child,(o.subtreeFlags&1028)!==0&&e!==null)e.return=o,be=e;else for(;be!==null;){o=be;try{var Re=o.alternate;if((o.flags&1024)!==0)switch(o.tag){case 0:case 11:case 15:break;case 1:if(Re!==null){var Pe=Re.memoizedProps,Pt=Re.memoizedState,q=o.stateNode,V=q.getSnapshotBeforeUpdate(o.elementType===o.type?Pe:_r(o.type,Pe),Pt);q.__reactInternalSnapshotBeforeUpdate=V}break;case 3:var X=o.stateNode.containerInfo;X.nodeType===1?X.textContent="":X.nodeType===9&&X.documentElement&&X.removeChild(X.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(n(163))}}catch(de){St(o,o.return,de)}if(e=o.sibling,e!==null){e.return=o.return,be=e;break}be=o.return}return Re=z0,z0=!1,Re}function Na(e,o,l){var d=o.updateQueue;if(d=d!==null?d.lastEffect:null,d!==null){var m=d=d.next;do{if((m.tag&e)===e){var v=m.destroy;m.destroy=void 0,v!==void 0&&iu(o,l,v)}m=m.next}while(m!==d)}}function Ji(e,o){if(o=o.updateQueue,o=o!==null?o.lastEffect:null,o!==null){var l=o=o.next;do{if((l.tag&e)===e){var d=l.create;l.destroy=d()}l=l.next}while(l!==o)}}function su(e){var o=e.ref;if(o!==null){var l=e.stateNode;switch(e.tag){case 5:e=l;break;default:e=l}typeof o=="function"?o(e):o.current=e}}function H0(e){var o=e.alternate;o!==null&&(e.alternate=null,H0(o)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(o=e.stateNode,o!==null&&(delete o[$r],delete o[ba],delete o[bc],delete o[ov],delete o[av])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function V0(e){return e.tag===5||e.tag===3||e.tag===4}function W0(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||V0(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function lu(e,o,l){var d=e.tag;if(d===5||d===6)e=e.stateNode,o?l.nodeType===8?l.parentNode.insertBefore(e,o):l.insertBefore(e,o):(l.nodeType===8?(o=l.parentNode,o.insertBefore(e,l)):(o=l,o.appendChild(e)),l=l._reactRootContainer,l!=null||o.onclick!==null||(o.onclick=Di));else if(d!==4&&(e=e.child,e!==null))for(lu(e,o,l),e=e.sibling;e!==null;)lu(e,o,l),e=e.sibling}function cu(e,o,l){var d=e.tag;if(d===5||d===6)e=e.stateNode,o?l.insertBefore(e,o):l.appendChild(e);else if(d!==4&&(e=e.child,e!==null))for(cu(e,o,l),e=e.sibling;e!==null;)cu(e,o,l),e=e.sibling}var Ut=null,Lr=!1;function Bn(e,o,l){for(l=l.child;l!==null;)G0(e,o,l),l=l.sibling}function G0(e,o,l){if(Qt&&typeof Qt.onCommitFiberUnmount=="function")try{Qt.onCommitFiberUnmount(Yr,l)}catch{}switch(l.tag){case 5:Kt||Ko(l,o);case 6:var d=Ut,m=Lr;Ut=null,Bn(e,o,l),Ut=d,Lr=m,Ut!==null&&(Lr?(e=Ut,l=l.stateNode,e.nodeType===8?e.parentNode.removeChild(l):e.removeChild(l)):Ut.removeChild(l.stateNode));break;case 18:Ut!==null&&(Lr?(e=Ut,l=l.stateNode,e.nodeType===8?Cc(e.parentNode,l):e.nodeType===1&&Cc(e,l),fa(e)):Cc(Ut,l.stateNode));break;case 4:d=Ut,m=Lr,Ut=l.stateNode.containerInfo,Lr=!0,Bn(e,o,l),Ut=d,Lr=m;break;case 0:case 11:case 14:case 15:if(!Kt&&(d=l.updateQueue,d!==null&&(d=d.lastEffect,d!==null))){m=d=d.next;do{var v=m,R=v.destroy;v=v.tag,R!==void 0&&((v&2)!==0||(v&4)!==0)&&iu(l,o,R),m=m.next}while(m!==d)}Bn(e,o,l);break;case 1:if(!Kt&&(Ko(l,o),d=l.stateNode,typeof d.componentWillUnmount=="function"))try{d.props=l.memoizedProps,d.state=l.memoizedState,d.componentWillUnmount()}catch(j){St(l,o,j)}Bn(e,o,l);break;case 21:Bn(e,o,l);break;case 22:l.mode&1?(Kt=(d=Kt)||l.memoizedState!==null,Bn(e,o,l),Kt=d):Bn(e,o,l);break;default:Bn(e,o,l)}}function q0(e){var o=e.updateQueue;if(o!==null){e.updateQueue=null;var l=e.stateNode;l===null&&(l=e.stateNode=new Ev),o.forEach(function(d){var m=_v.bind(null,e,d);l.has(d)||(l.add(d),d.then(m,m))})}}function Fr(e,o){var l=o.deletions;if(l!==null)for(var d=0;d<l.length;d++){var m=l[d];try{var v=e,R=o,j=R;e:for(;j!==null;){switch(j.tag){case 5:Ut=j.stateNode,Lr=!1;break e;case 3:Ut=j.stateNode.containerInfo,Lr=!0;break e;case 4:Ut=j.stateNode.containerInfo,Lr=!0;break e}j=j.return}if(Ut===null)throw Error(n(160));G0(v,R,m),Ut=null,Lr=!1;var U=m.alternate;U!==null&&(U.return=null),m.return=null}catch(J){St(m,o,J)}}if(o.subtreeFlags&12854)for(o=o.child;o!==null;)K0(o,e),o=o.sibling}function K0(e,o){var l=e.alternate,d=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(Fr(o,e),Hr(e),d&4){try{Na(3,e,e.return),Ji(3,e)}catch(Pe){St(e,e.return,Pe)}try{Na(5,e,e.return)}catch(Pe){St(e,e.return,Pe)}}break;case 1:Fr(o,e),Hr(e),d&512&&l!==null&&Ko(l,l.return);break;case 5:if(Fr(o,e),Hr(e),d&512&&l!==null&&Ko(l,l.return),e.flags&32){var m=e.stateNode;try{Tt(m,"")}catch(Pe){St(e,e.return,Pe)}}if(d&4&&(m=e.stateNode,m!=null)){var v=e.memoizedProps,R=l!==null?l.memoizedProps:v,j=e.type,U=e.updateQueue;if(e.updateQueue=null,U!==null)try{j==="input"&&v.type==="radio"&&v.name!=null&&Ye(m,v),gn(j,R);var J=gn(j,v);for(R=0;R<U.length;R+=2){var le=U[R],ce=U[R+1];le==="style"?Yt(m,ce):le==="dangerouslySetInnerHTML"?mt(m,ce):le==="children"?Tt(m,ce):A(m,le,ce,J)}switch(j){case"input":tt(m,v);break;case"textarea":Oe(m,v);break;case"select":var ie=m._wrapperState.wasMultiple;m._wrapperState.wasMultiple=!!v.multiple;var we=v.value;we!=null?ye(m,!!v.multiple,we,!1):ie!==!!v.multiple&&(v.defaultValue!=null?ye(m,!!v.multiple,v.defaultValue,!0):ye(m,!!v.multiple,v.multiple?[]:"",!1))}m[ba]=v}catch(Pe){St(e,e.return,Pe)}}break;case 6:if(Fr(o,e),Hr(e),d&4){if(e.stateNode===null)throw Error(n(162));m=e.stateNode,v=e.memoizedProps;try{m.nodeValue=v}catch(Pe){St(e,e.return,Pe)}}break;case 3:if(Fr(o,e),Hr(e),d&4&&l!==null&&l.memoizedState.isDehydrated)try{fa(o.containerInfo)}catch(Pe){St(e,e.return,Pe)}break;case 4:Fr(o,e),Hr(e);break;case 13:Fr(o,e),Hr(e),m=e.child,m.flags&8192&&(v=m.memoizedState!==null,m.stateNode.isHidden=v,!v||m.alternate!==null&&m.alternate.memoizedState!==null||(fu=at())),d&4&&q0(e);break;case 22:if(le=l!==null&&l.memoizedState!==null,e.mode&1?(Kt=(J=Kt)||le,Fr(o,e),Kt=J):Fr(o,e),Hr(e),d&8192){if(J=e.memoizedState!==null,(e.stateNode.isHidden=J)&&!le&&(e.mode&1)!==0)for(be=e,le=e.child;le!==null;){for(ce=be=le;be!==null;){switch(ie=be,we=ie.child,ie.tag){case 0:case 11:case 14:case 15:Na(4,ie,ie.return);break;case 1:Ko(ie,ie.return);var Re=ie.stateNode;if(typeof Re.componentWillUnmount=="function"){d=ie,l=ie.return;try{o=d,Re.props=o.memoizedProps,Re.state=o.memoizedState,Re.componentWillUnmount()}catch(Pe){St(d,l,Pe)}}break;case 5:Ko(ie,ie.return);break;case 22:if(ie.memoizedState!==null){Q0(ce);continue}}we!==null?(we.return=ie,be=we):Q0(ce)}le=le.sibling}e:for(le=null,ce=e;;){if(ce.tag===5){if(le===null){le=ce;try{m=ce.stateNode,J?(v=m.style,typeof v.setProperty=="function"?v.setProperty("display","none","important"):v.display="none"):(j=ce.stateNode,U=ce.memoizedProps.style,R=U!=null&&U.hasOwnProperty("display")?U.display:null,j.style.display=Po("display",R))}catch(Pe){St(e,e.return,Pe)}}}else if(ce.tag===6){if(le===null)try{ce.stateNode.nodeValue=J?"":ce.memoizedProps}catch(Pe){St(e,e.return,Pe)}}else if((ce.tag!==22&&ce.tag!==23||ce.memoizedState===null||ce===e)&&ce.child!==null){ce.child.return=ce,ce=ce.child;continue}if(ce===e)break e;for(;ce.sibling===null;){if(ce.return===null||ce.return===e)break e;le===ce&&(le=null),ce=ce.return}le===ce&&(le=null),ce.sibling.return=ce.return,ce=ce.sibling}}break;case 19:Fr(o,e),Hr(e),d&4&&q0(e);break;case 21:break;default:Fr(o,e),Hr(e)}}function Hr(e){var o=e.flags;if(o&2){try{e:{for(var l=e.return;l!==null;){if(V0(l)){var d=l;break e}l=l.return}throw Error(n(160))}switch(d.tag){case 5:var m=d.stateNode;d.flags&32&&(Tt(m,""),d.flags&=-33);var v=W0(e);cu(e,v,m);break;case 3:case 4:var R=d.stateNode.containerInfo,j=W0(e);lu(e,j,R);break;default:throw Error(n(161))}}catch(U){St(e,e.return,U)}e.flags&=-3}o&4096&&(e.flags&=-4097)}function bv(e,o,l){be=e,X0(e)}function X0(e,o,l){for(var d=(e.mode&1)!==0;be!==null;){var m=be,v=m.child;if(m.tag===22&&d){var R=m.memoizedState!==null||Qi;if(!R){var j=m.alternate,U=j!==null&&j.memoizedState!==null||Kt;j=Qi;var J=Kt;if(Qi=R,(Kt=U)&&!J)for(be=m;be!==null;)R=be,U=R.child,R.tag===22&&R.memoizedState!==null?J0(m):U!==null?(U.return=R,be=U):J0(m);for(;v!==null;)be=v,X0(v),v=v.sibling;be=m,Qi=j,Kt=J}Y0(e)}else(m.subtreeFlags&8772)!==0&&v!==null?(v.return=m,be=v):Y0(e)}}function Y0(e){for(;be!==null;){var o=be;if((o.flags&8772)!==0){var l=o.alternate;try{if((o.flags&8772)!==0)switch(o.tag){case 0:case 11:case 15:Kt||Ji(5,o);break;case 1:var d=o.stateNode;if(o.flags&4&&!Kt)if(l===null)d.componentDidMount();else{var m=o.elementType===o.type?l.memoizedProps:_r(o.type,l.memoizedProps);d.componentDidUpdate(m,l.memoizedState,d.__reactInternalSnapshotBeforeUpdate)}var v=o.updateQueue;v!==null&&Qf(o,v,d);break;case 3:var R=o.updateQueue;if(R!==null){if(l=null,o.child!==null)switch(o.child.tag){case 5:l=o.child.stateNode;break;case 1:l=o.child.stateNode}Qf(o,R,l)}break;case 5:var j=o.stateNode;if(l===null&&o.flags&4){l=j;var U=o.memoizedProps;switch(o.type){case"button":case"input":case"select":case"textarea":U.autoFocus&&l.focus();break;case"img":U.src&&(l.src=U.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(o.memoizedState===null){var J=o.alternate;if(J!==null){var le=J.memoizedState;if(le!==null){var ce=le.dehydrated;ce!==null&&fa(ce)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(n(163))}Kt||o.flags&512&&su(o)}catch(ie){St(o,o.return,ie)}}if(o===e){be=null;break}if(l=o.sibling,l!==null){l.return=o.return,be=l;break}be=o.return}}function Q0(e){for(;be!==null;){var o=be;if(o===e){be=null;break}var l=o.sibling;if(l!==null){l.return=o.return,be=l;break}be=o.return}}function J0(e){for(;be!==null;){var o=be;try{switch(o.tag){case 0:case 11:case 15:var l=o.return;try{Ji(4,o)}catch(U){St(o,l,U)}break;case 1:var d=o.stateNode;if(typeof d.componentDidMount=="function"){var m=o.return;try{d.componentDidMount()}catch(U){St(o,m,U)}}var v=o.return;try{su(o)}catch(U){St(o,v,U)}break;case 5:var R=o.return;try{su(o)}catch(U){St(o,R,U)}}}catch(U){St(o,o.return,U)}if(o===e){be=null;break}var j=o.sibling;if(j!==null){j.return=o.return,be=j;break}be=o.return}}var Sv=Math.ceil,Zi=T.ReactCurrentDispatcher,uu=T.ReactCurrentOwner,Cr=T.ReactCurrentBatchConfig,Ze=0,Ot=null,Lt=null,zt=0,mr=0,Xo=_n(0),It=0,Ia=null,po=0,es=0,du=0,Ba=null,ar=null,fu=0,Yo=1/0,sn=null,ts=!1,pu=null,jn=null,rs=!1,On=null,ns=0,ja=0,hu=null,os=-1,as=0;function Zt(){return(Ze&6)!==0?at():os!==-1?os:os=at()}function Mn(e){return(e.mode&1)===0?1:(Ze&2)!==0&&zt!==0?zt&-zt:sv.transition!==null?(as===0&&(as=Vd()),as):(e=it,e!==0||(e=window.event,e=e===void 0?16:Zd(e.type)),e)}function Nr(e,o,l,d){if(50<ja)throw ja=0,hu=null,Error(n(185));sa(e,l,d),((Ze&2)===0||e!==Ot)&&(e===Ot&&((Ze&2)===0&&(es|=l),It===4&&$n(e,zt)),ir(e,d),l===1&&Ze===0&&(o.mode&1)===0&&(Yo=at()+500,Fi&&Fn()))}function ir(e,o){var l=e.callbackNode;sx(e,o);var d=hi(e,e===Ot?zt:0);if(d===0)l!==null&&Bt(l),e.callbackNode=null,e.callbackPriority=0;else if(o=d&-d,e.callbackPriority!==o){if(l!=null&&Bt(l),o===1)e.tag===0?iv(ep.bind(null,e)):Mf(ep.bind(null,e)),rv(function(){(Ze&6)===0&&Fn()}),l=null;else{switch(Wd(d)){case 1:l=xr;break;case 4:l=dr;break;case 16:l=Cn;break;case 536870912:l=Mr;break;default:l=Cn}l=lp(l,Z0.bind(null,e))}e.callbackPriority=o,e.callbackNode=l}}function Z0(e,o){if(os=-1,as=0,(Ze&6)!==0)throw Error(n(327));var l=e.callbackNode;if(Qo()&&e.callbackNode!==l)return null;var d=hi(e,e===Ot?zt:0);if(d===0)return null;if((d&30)!==0||(d&e.expiredLanes)!==0||o)o=is(e,d);else{o=d;var m=Ze;Ze|=2;var v=rp();(Ot!==e||zt!==o)&&(sn=null,Yo=at()+500,mo(e,o));do try{Pv();break}catch(j){tp(e,j)}while(!0);Lc(),Zi.current=v,Ze=m,Lt!==null?o=0:(Ot=null,zt=0,o=It)}if(o!==0){if(o===2&&(m=Kl(e),m!==0&&(d=m,o=mu(e,m))),o===1)throw l=Ia,mo(e,0),$n(e,d),ir(e,at()),l;if(o===6)$n(e,d);else{if(m=e.current.alternate,(d&30)===0&&!Rv(m)&&(o=is(e,d),o===2&&(v=Kl(e),v!==0&&(d=v,o=mu(e,v))),o===1))throw l=Ia,mo(e,0),$n(e,d),ir(e,at()),l;switch(e.finishedWork=m,e.finishedLanes=d,o){case 0:case 1:throw Error(n(345));case 2:go(e,ar,sn);break;case 3:if($n(e,d),(d&130023424)===d&&(o=fu+500-at(),10<o)){if(hi(e,0)!==0)break;if(m=e.suspendedLanes,(m&d)!==d){Zt(),e.pingedLanes|=e.suspendedLanes&m;break}e.timeoutHandle=Ec(go.bind(null,e,ar,sn),o);break}go(e,ar,sn);break;case 4:if($n(e,d),(d&4194240)===d)break;for(o=e.eventTimes,m=-1;0<d;){var R=31-ut(d);v=1<<R,R=o[R],R>m&&(m=R),d&=~v}if(d=m,d=at()-d,d=(120>d?120:480>d?480:1080>d?1080:1920>d?1920:3e3>d?3e3:4320>d?4320:1960*Sv(d/1960))-d,10<d){e.timeoutHandle=Ec(go.bind(null,e,ar,sn),d);break}go(e,ar,sn);break;case 5:go(e,ar,sn);break;default:throw Error(n(329))}}}return ir(e,at()),e.callbackNode===l?Z0.bind(null,e):null}function mu(e,o){var l=Ba;return e.current.memoizedState.isDehydrated&&(mo(e,o).flags|=256),e=is(e,o),e!==2&&(o=ar,ar=l,o!==null&&gu(o)),e}function gu(e){ar===null?ar=e:ar.push.apply(ar,e)}function Rv(e){for(var o=e;;){if(o.flags&16384){var l=o.updateQueue;if(l!==null&&(l=l.stores,l!==null))for(var d=0;d<l.length;d++){var m=l[d],v=m.getSnapshot;m=m.value;try{if(!Dr(v(),m))return!1}catch{return!1}}}if(l=o.child,o.subtreeFlags&16384&&l!==null)l.return=o,o=l;else{if(o===e)break;for(;o.sibling===null;){if(o.return===null||o.return===e)return!0;o=o.return}o.sibling.return=o.return,o=o.sibling}}return!0}function $n(e,o){for(o&=~du,o&=~es,e.suspendedLanes|=o,e.pingedLanes&=~o,e=e.expirationTimes;0<o;){var l=31-ut(o),d=1<<l;e[l]=-1,o&=~d}}function ep(e){if((Ze&6)!==0)throw Error(n(327));Qo();var o=hi(e,0);if((o&1)===0)return ir(e,at()),null;var l=is(e,o);if(e.tag!==0&&l===2){var d=Kl(e);d!==0&&(o=d,l=mu(e,d))}if(l===1)throw l=Ia,mo(e,0),$n(e,o),ir(e,at()),l;if(l===6)throw Error(n(345));return e.finishedWork=e.current.alternate,e.finishedLanes=o,go(e,ar,sn),ir(e,at()),null}function xu(e,o){var l=Ze;Ze|=1;try{return e(o)}finally{Ze=l,Ze===0&&(Yo=at()+500,Fi&&Fn())}}function ho(e){On!==null&&On.tag===0&&(Ze&6)===0&&Qo();var o=Ze;Ze|=1;var l=Cr.transition,d=it;try{if(Cr.transition=null,it=1,e)return e()}finally{it=d,Cr.transition=l,Ze=o,(Ze&6)===0&&Fn()}}function vu(){mr=Xo.current,xt(Xo)}function mo(e,o){e.finishedWork=null,e.finishedLanes=0;var l=e.timeoutHandle;if(l!==-1&&(e.timeoutHandle=-1,tv(l)),Lt!==null)for(l=Lt.return;l!==null;){var d=l;switch(Pc(d),d.tag){case 1:d=d.type.childContextTypes,d!=null&&_i();break;case 3:Go(),xt(rr),xt(Wt),$c();break;case 5:Oc(d);break;case 4:Go();break;case 13:xt(wt);break;case 19:xt(wt);break;case 10:Fc(d.type._context);break;case 22:case 23:vu()}l=l.return}if(Ot=e,Lt=e=Un(e.current,null),zt=mr=o,It=0,Ia=null,du=es=po=0,ar=Ba=null,co!==null){for(o=0;o<co.length;o++)if(l=co[o],d=l.interleaved,d!==null){l.interleaved=null;var m=d.next,v=l.pending;if(v!==null){var R=v.next;v.next=m,d.next=R}l.pending=d}co=null}return e}function tp(e,o){do{var l=Lt;try{if(Lc(),Hi.current=qi,Vi){for(var d=Et.memoizedState;d!==null;){var m=d.queue;m!==null&&(m.pending=null),d=d.next}Vi=!1}if(fo=0,jt=Nt=Et=null,Da=!1,Ta=0,uu.current=null,l===null||l.return===null){It=1,Ia=o,Lt=null;break}e:{var v=e,R=l.return,j=l,U=o;if(o=zt,j.flags|=32768,U!==null&&typeof U=="object"&&typeof U.then=="function"){var J=U,le=j,ce=le.tag;if((le.mode&1)===0&&(ce===0||ce===11||ce===15)){var ie=le.alternate;ie?(le.updateQueue=ie.updateQueue,le.memoizedState=ie.memoizedState,le.lanes=ie.lanes):(le.updateQueue=null,le.memoizedState=null)}var we=A0(R);if(we!==null){we.flags&=-257,P0(we,R,j,v,o),we.mode&1&&R0(v,J,o),o=we,U=J;var Re=o.updateQueue;if(Re===null){var Pe=new Set;Pe.add(U),o.updateQueue=Pe}else Re.add(U);break e}else{if((o&1)===0){R0(v,J,o),yu();break e}U=Error(n(426))}}else if(yt&&j.mode&1){var Pt=A0(R);if(Pt!==null){(Pt.flags&65536)===0&&(Pt.flags|=256),P0(Pt,R,j,v,o),Tc(qo(U,j));break e}}v=U=qo(U,j),It!==4&&(It=2),Ba===null?Ba=[v]:Ba.push(v),v=R;do{switch(v.tag){case 3:v.flags|=65536,o&=-o,v.lanes|=o;var q=b0(v,U,o);Yf(v,q);break e;case 1:j=U;var V=v.type,X=v.stateNode;if((v.flags&128)===0&&(typeof V.getDerivedStateFromError=="function"||X!==null&&typeof X.componentDidCatch=="function"&&(jn===null||!jn.has(X)))){v.flags|=65536,o&=-o,v.lanes|=o;var de=S0(v,j,o);Yf(v,de);break e}}v=v.return}while(v!==null)}op(l)}catch(Te){o=Te,Lt===l&&l!==null&&(Lt=l=l.return);continue}break}while(!0)}function rp(){var e=Zi.current;return Zi.current=qi,e===null?qi:e}function yu(){(It===0||It===3||It===2)&&(It=4),Ot===null||(po&268435455)===0&&(es&268435455)===0||$n(Ot,zt)}function is(e,o){var l=Ze;Ze|=2;var d=rp();(Ot!==e||zt!==o)&&(sn=null,mo(e,o));do try{Av();break}catch(m){tp(e,m)}while(!0);if(Lc(),Ze=l,Zi.current=d,Lt!==null)throw Error(n(261));return Ot=null,zt=0,It}function Av(){for(;Lt!==null;)np(Lt)}function Pv(){for(;Lt!==null&&!nt();)np(Lt)}function np(e){var o=sp(e.alternate,e,mr);e.memoizedProps=e.pendingProps,o===null?op(e):Lt=o,uu.current=null}function op(e){var o=e;do{var l=o.alternate;if(e=o.return,(o.flags&32768)===0){if(l=yv(l,o,mr),l!==null){Lt=l;return}}else{if(l=wv(l,o),l!==null){l.flags&=32767,Lt=l;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{It=6,Lt=null;return}}if(o=o.sibling,o!==null){Lt=o;return}Lt=o=e}while(o!==null);It===0&&(It=5)}function go(e,o,l){var d=it,m=Cr.transition;try{Cr.transition=null,it=1,kv(e,o,l,d)}finally{Cr.transition=m,it=d}return null}function kv(e,o,l,d){do Qo();while(On!==null);if((Ze&6)!==0)throw Error(n(327));l=e.finishedWork;var m=e.finishedLanes;if(l===null)return null;if(e.finishedWork=null,e.finishedLanes=0,l===e.current)throw Error(n(177));e.callbackNode=null,e.callbackPriority=0;var v=l.lanes|l.childLanes;if(lx(e,v),e===Ot&&(Lt=Ot=null,zt=0),(l.subtreeFlags&2064)===0&&(l.flags&2064)===0||rs||(rs=!0,lp(Cn,function(){return Qo(),null})),v=(l.flags&15990)!==0,(l.subtreeFlags&15990)!==0||v){v=Cr.transition,Cr.transition=null;var R=it;it=1;var j=Ze;Ze|=4,uu.current=null,Cv(e,l),K0(l,e),Kx(yc),xi=!!vc,yc=vc=null,e.current=l,bv(l),tr(),Ze=j,it=R,Cr.transition=v}else e.current=l;if(rs&&(rs=!1,On=e,ns=m),v=e.pendingLanes,v===0&&(jn=null),Qe(l.stateNode),ir(e,at()),o!==null)for(d=e.onRecoverableError,l=0;l<o.length;l++)m=o[l],d(m.value,{componentStack:m.stack,digest:m.digest});if(ts)throw ts=!1,e=pu,pu=null,e;return(ns&1)!==0&&e.tag!==0&&Qo(),v=e.pendingLanes,(v&1)!==0?e===hu?ja++:(ja=0,hu=e):ja=0,Fn(),null}function Qo(){if(On!==null){var e=Wd(ns),o=Cr.transition,l=it;try{if(Cr.transition=null,it=16>e?16:e,On===null)var d=!1;else{if(e=On,On=null,ns=0,(Ze&6)!==0)throw Error(n(331));var m=Ze;for(Ze|=4,be=e.current;be!==null;){var v=be,R=v.child;if((be.flags&16)!==0){var j=v.deletions;if(j!==null){for(var U=0;U<j.length;U++){var J=j[U];for(be=J;be!==null;){var le=be;switch(le.tag){case 0:case 11:case 15:Na(8,le,v)}var ce=le.child;if(ce!==null)ce.return=le,be=ce;else for(;be!==null;){le=be;var ie=le.sibling,we=le.return;if(H0(le),le===J){be=null;break}if(ie!==null){ie.return=we,be=ie;break}be=we}}}var Re=v.alternate;if(Re!==null){var Pe=Re.child;if(Pe!==null){Re.child=null;do{var Pt=Pe.sibling;Pe.sibling=null,Pe=Pt}while(Pe!==null)}}be=v}}if((v.subtreeFlags&2064)!==0&&R!==null)R.return=v,be=R;else e:for(;be!==null;){if(v=be,(v.flags&2048)!==0)switch(v.tag){case 0:case 11:case 15:Na(9,v,v.return)}var q=v.sibling;if(q!==null){q.return=v.return,be=q;break e}be=v.return}}var V=e.current;for(be=V;be!==null;){R=be;var X=R.child;if((R.subtreeFlags&2064)!==0&&X!==null)X.return=R,be=X;else e:for(R=V;be!==null;){if(j=be,(j.flags&2048)!==0)try{switch(j.tag){case 0:case 11:case 15:Ji(9,j)}}catch(Te){St(j,j.return,Te)}if(j===R){be=null;break e}var de=j.sibling;if(de!==null){de.return=j.return,be=de;break e}be=j.return}}if(Ze=m,Fn(),Qt&&typeof Qt.onPostCommitFiberRoot=="function")try{Qt.onPostCommitFiberRoot(Yr,e)}catch{}d=!0}return d}finally{it=l,Cr.transition=o}}return!1}function ap(e,o,l){o=qo(l,o),o=b0(e,o,1),e=In(e,o,1),o=Zt(),e!==null&&(sa(e,1,o),ir(e,o))}function St(e,o,l){if(e.tag===3)ap(e,e,l);else for(;o!==null;){if(o.tag===3){ap(o,e,l);break}else if(o.tag===1){var d=o.stateNode;if(typeof o.type.getDerivedStateFromError=="function"||typeof d.componentDidCatch=="function"&&(jn===null||!jn.has(d))){e=qo(l,e),e=S0(o,e,1),o=In(o,e,1),e=Zt(),o!==null&&(sa(o,1,e),ir(o,e));break}}o=o.return}}function Dv(e,o,l){var d=e.pingCache;d!==null&&d.delete(o),o=Zt(),e.pingedLanes|=e.suspendedLanes&l,Ot===e&&(zt&l)===l&&(It===4||It===3&&(zt&130023424)===zt&&500>at()-fu?mo(e,0):du|=l),ir(e,o)}function ip(e,o){o===0&&((e.mode&1)===0?o=1:(o=no,no<<=1,(no&130023424)===0&&(no=4194304)));var l=Zt();e=nn(e,o),e!==null&&(sa(e,o,l),ir(e,l))}function Tv(e){var o=e.memoizedState,l=0;o!==null&&(l=o.retryLane),ip(e,l)}function _v(e,o){var l=0;switch(e.tag){case 13:var d=e.stateNode,m=e.memoizedState;m!==null&&(l=m.retryLane);break;case 19:d=e.stateNode;break;default:throw Error(n(314))}d!==null&&d.delete(o),ip(e,l)}var sp;sp=function(e,o,l){if(e!==null)if(e.memoizedProps!==o.pendingProps||rr.current)or=!0;else{if((e.lanes&l)===0&&(o.flags&128)===0)return or=!1,vv(e,o,l);or=(e.flags&131072)!==0}else or=!1,yt&&(o.flags&1048576)!==0&&$f(o,Ii,o.index);switch(o.lanes=0,o.tag){case 2:var d=o.type;Yi(e,o),e=o.pendingProps;var m=Mo(o,Wt.current);Wo(o,l),m=Hc(null,o,d,e,m,l);var v=Vc();return o.flags|=1,typeof m=="object"&&m!==null&&typeof m.render=="function"&&m.$$typeof===void 0?(o.tag=1,o.memoizedState=null,o.updateQueue=null,nr(d)?(v=!0,Li(o)):v=!1,o.memoizedState=m.state!==null&&m.state!==void 0?m.state:null,Bc(o),m.updater=Ki,o.stateNode=m,m._reactInternals=o,Yc(o,d,e,l),o=eu(null,o,d,!0,v,l)):(o.tag=0,yt&&v&&Ac(o),Jt(null,o,m,l),o=o.child),o;case 16:d=o.elementType;e:{switch(Yi(e,o),e=o.pendingProps,m=d._init,d=m(d._payload),o.type=d,m=o.tag=Fv(d),e=_r(d,e),m){case 0:o=Zc(null,o,d,e,l);break e;case 1:o=F0(null,o,d,e,l);break e;case 11:o=k0(null,o,d,e,l);break e;case 14:o=D0(null,o,d,_r(d.type,e),l);break e}throw Error(n(306,d,""))}return o;case 0:return d=o.type,m=o.pendingProps,m=o.elementType===d?m:_r(d,m),Zc(e,o,d,m,l);case 1:return d=o.type,m=o.pendingProps,m=o.elementType===d?m:_r(d,m),F0(e,o,d,m,l);case 3:e:{if(N0(o),e===null)throw Error(n(387));d=o.pendingProps,v=o.memoizedState,m=v.element,Xf(e,o),Ui(o,d,null,l);var R=o.memoizedState;if(d=R.element,v.isDehydrated)if(v={element:d,isDehydrated:!1,cache:R.cache,pendingSuspenseBoundaries:R.pendingSuspenseBoundaries,transitions:R.transitions},o.updateQueue.baseState=v,o.memoizedState=v,o.flags&256){m=qo(Error(n(423)),o),o=I0(e,o,d,l,m);break e}else if(d!==m){m=qo(Error(n(424)),o),o=I0(e,o,d,l,m);break e}else for(hr=Tn(o.stateNode.containerInfo.firstChild),pr=o,yt=!0,Tr=null,l=qf(o,null,d,l),o.child=l;l;)l.flags=l.flags&-3|4096,l=l.sibling;else{if(zo(),d===m){o=an(e,o,l);break e}Jt(e,o,d,l)}o=o.child}return o;case 5:return Jf(o),e===null&&Dc(o),d=o.type,m=o.pendingProps,v=e!==null?e.memoizedProps:null,R=m.children,wc(d,m)?R=null:v!==null&&wc(d,v)&&(o.flags|=32),L0(e,o),Jt(e,o,R,l),o.child;case 6:return e===null&&Dc(o),null;case 13:return B0(e,o,l);case 4:return jc(o,o.stateNode.containerInfo),d=o.pendingProps,e===null?o.child=Ho(o,null,d,l):Jt(e,o,d,l),o.child;case 11:return d=o.type,m=o.pendingProps,m=o.elementType===d?m:_r(d,m),k0(e,o,d,m,l);case 7:return Jt(e,o,o.pendingProps,l),o.child;case 8:return Jt(e,o,o.pendingProps.children,l),o.child;case 12:return Jt(e,o,o.pendingProps.children,l),o.child;case 10:e:{if(d=o.type._context,m=o.pendingProps,v=o.memoizedProps,R=m.value,pt(Oi,d._currentValue),d._currentValue=R,v!==null)if(Dr(v.value,R)){if(v.children===m.children&&!rr.current){o=an(e,o,l);break e}}else for(v=o.child,v!==null&&(v.return=o);v!==null;){var j=v.dependencies;if(j!==null){R=v.child;for(var U=j.firstContext;U!==null;){if(U.context===d){if(v.tag===1){U=on(-1,l&-l),U.tag=2;var J=v.updateQueue;if(J!==null){J=J.shared;var le=J.pending;le===null?U.next=U:(U.next=le.next,le.next=U),J.pending=U}}v.lanes|=l,U=v.alternate,U!==null&&(U.lanes|=l),Nc(v.return,l,o),j.lanes|=l;break}U=U.next}}else if(v.tag===10)R=v.type===o.type?null:v.child;else if(v.tag===18){if(R=v.return,R===null)throw Error(n(341));R.lanes|=l,j=R.alternate,j!==null&&(j.lanes|=l),Nc(R,l,o),R=v.sibling}else R=v.child;if(R!==null)R.return=v;else for(R=v;R!==null;){if(R===o){R=null;break}if(v=R.sibling,v!==null){v.return=R.return,R=v;break}R=R.return}v=R}Jt(e,o,m.children,l),o=o.child}return o;case 9:return m=o.type,d=o.pendingProps.children,Wo(o,l),m=wr(m),d=d(m),o.flags|=1,Jt(e,o,d,l),o.child;case 14:return d=o.type,m=_r(d,o.pendingProps),m=_r(d.type,m),D0(e,o,d,m,l);case 15:return T0(e,o,o.type,o.pendingProps,l);case 17:return d=o.type,m=o.pendingProps,m=o.elementType===d?m:_r(d,m),Yi(e,o),o.tag=1,nr(d)?(e=!0,Li(o)):e=!1,Wo(o,l),E0(o,d,m),Yc(o,d,m,l),eu(null,o,d,!0,e,l);case 19:return O0(e,o,l);case 22:return _0(e,o,l)}throw Error(n(156,o.tag))};function lp(e,o){return At(e,o)}function Lv(e,o,l,d){this.tag=e,this.key=l,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=o,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=d,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function br(e,o,l,d){return new Lv(e,o,l,d)}function wu(e){return e=e.prototype,!(!e||!e.isReactComponent)}function Fv(e){if(typeof e=="function")return wu(e)?1:0;if(e!=null){if(e=e.$$typeof,e===H)return 11;if(e===oe)return 14}return 2}function Un(e,o){var l=e.alternate;return l===null?(l=br(e.tag,o,e.key,e.mode),l.elementType=e.elementType,l.type=e.type,l.stateNode=e.stateNode,l.alternate=e,e.alternate=l):(l.pendingProps=o,l.type=e.type,l.flags=0,l.subtreeFlags=0,l.deletions=null),l.flags=e.flags&14680064,l.childLanes=e.childLanes,l.lanes=e.lanes,l.child=e.child,l.memoizedProps=e.memoizedProps,l.memoizedState=e.memoizedState,l.updateQueue=e.updateQueue,o=e.dependencies,l.dependencies=o===null?null:{lanes:o.lanes,firstContext:o.firstContext},l.sibling=e.sibling,l.index=e.index,l.ref=e.ref,l}function ss(e,o,l,d,m,v){var R=2;if(d=e,typeof e=="function")wu(e)&&(R=1);else if(typeof e=="string")R=5;else e:switch(e){case k:return xo(l.children,m,v,o);case N:R=8,m|=8;break;case I:return e=br(12,l,o,m|2),e.elementType=I,e.lanes=v,e;case z:return e=br(13,l,o,m),e.elementType=z,e.lanes=v,e;case Z:return e=br(19,l,o,m),e.elementType=Z,e.lanes=v,e;case se:return ls(l,m,v,o);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case L:R=10;break e;case M:R=9;break e;case H:R=11;break e;case oe:R=14;break e;case re:R=16,d=null;break e}throw Error(n(130,e==null?e:typeof e,""))}return o=br(R,l,o,m),o.elementType=e,o.type=d,o.lanes=v,o}function xo(e,o,l,d){return e=br(7,e,d,o),e.lanes=l,e}function ls(e,o,l,d){return e=br(22,e,d,o),e.elementType=se,e.lanes=l,e.stateNode={isHidden:!1},e}function Eu(e,o,l){return e=br(6,e,null,o),e.lanes=l,e}function Cu(e,o,l){return o=br(4,e.children!==null?e.children:[],e.key,o),o.lanes=l,o.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},o}function Nv(e,o,l,d,m){this.tag=o,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Xl(0),this.expirationTimes=Xl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Xl(0),this.identifierPrefix=d,this.onRecoverableError=m,this.mutableSourceEagerHydrationData=null}function bu(e,o,l,d,m,v,R,j,U){return e=new Nv(e,o,l,j,U),o===1?(o=1,v===!0&&(o|=8)):o=0,v=br(3,null,null,o),e.current=v,v.stateNode=e,v.memoizedState={element:d,isDehydrated:l,cache:null,transitions:null,pendingSuspenseBoundaries:null},Bc(v),e}function Iv(e,o,l){var d=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:F,key:d==null?null:""+d,children:e,containerInfo:o,implementation:l}}function cp(e){if(!e)return Ln;e=e._reactInternals;e:{if(Ae(e)!==e||e.tag!==1)throw Error(n(170));var o=e;do{switch(o.tag){case 3:o=o.stateNode.context;break e;case 1:if(nr(o.type)){o=o.stateNode.__reactInternalMemoizedMergedChildContext;break e}}o=o.return}while(o!==null);throw Error(n(171))}if(e.tag===1){var l=e.type;if(nr(l))return jf(e,l,o)}return o}function up(e,o,l,d,m,v,R,j,U){return e=bu(l,d,!0,e,m,v,R,j,U),e.context=cp(null),l=e.current,d=Zt(),m=Mn(l),v=on(d,m),v.callback=o??null,In(l,v,m),e.current.lanes=m,sa(e,m,d),ir(e,d),e}function cs(e,o,l,d){var m=o.current,v=Zt(),R=Mn(m);return l=cp(l),o.context===null?o.context=l:o.pendingContext=l,o=on(v,R),o.payload={element:e},d=d===void 0?null:d,d!==null&&(o.callback=d),e=In(m,o,R),e!==null&&(Nr(e,m,R,v),$i(e,m,R)),R}function us(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function dp(e,o){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var l=e.retryLane;e.retryLane=l!==0&&l<o?l:o}}function Su(e,o){dp(e,o),(e=e.alternate)&&dp(e,o)}function Bv(){return null}var fp=typeof reportError=="function"?reportError:function(e){console.error(e)};function Ru(e){this._internalRoot=e}ds.prototype.render=Ru.prototype.render=function(e){var o=this._internalRoot;if(o===null)throw Error(n(409));cs(e,o,null,null)},ds.prototype.unmount=Ru.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var o=e.containerInfo;ho(function(){cs(null,e,null,null)}),o[Zr]=null}};function ds(e){this._internalRoot=e}ds.prototype.unstable_scheduleHydration=function(e){if(e){var o=Kd();e={blockedOn:null,target:e,priority:o};for(var l=0;l<Pn.length&&o!==0&&o<Pn[l].priority;l++);Pn.splice(l,0,e),l===0&&Qd(e)}};function Au(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function fs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function pp(){}function jv(e,o,l,d,m){if(m){if(typeof d=="function"){var v=d;d=function(){var J=us(R);v.call(J)}}var R=up(o,d,e,0,null,!1,!1,"",pp);return e._reactRootContainer=R,e[Zr]=R.current,Ea(e.nodeType===8?e.parentNode:e),ho(),R}for(;m=e.lastChild;)e.removeChild(m);if(typeof d=="function"){var j=d;d=function(){var J=us(U);j.call(J)}}var U=bu(e,0,!1,null,null,!1,!1,"",pp);return e._reactRootContainer=U,e[Zr]=U.current,Ea(e.nodeType===8?e.parentNode:e),ho(function(){cs(o,U,l,d)}),U}function ps(e,o,l,d,m){var v=l._reactRootContainer;if(v){var R=v;if(typeof m=="function"){var j=m;m=function(){var U=us(R);j.call(U)}}cs(o,R,e,m)}else R=jv(l,o,e,m,d);return us(R)}Gd=function(e){switch(e.tag){case 3:var o=e.stateNode;if(o.current.memoizedState.isDehydrated){var l=oo(o.pendingLanes);l!==0&&(Yl(o,l|1),ir(o,at()),(Ze&6)===0&&(Yo=at()+500,Fn()))}break;case 13:ho(function(){var d=nn(e,1);if(d!==null){var m=Zt();Nr(d,e,1,m)}}),Su(e,1)}},Ql=function(e){if(e.tag===13){var o=nn(e,134217728);if(o!==null){var l=Zt();Nr(o,e,134217728,l)}Su(e,134217728)}},qd=function(e){if(e.tag===13){var o=Mn(e),l=nn(e,o);if(l!==null){var d=Zt();Nr(l,e,o,d)}Su(e,o)}},Kd=function(){return it},Xd=function(e,o){var l=it;try{return it=e,o()}finally{it=l}},Pr=function(e,o,l){switch(o){case"input":if(tt(e,l),o=l.name,l.type==="radio"&&o!=null){for(l=e;l.parentNode;)l=l.parentNode;for(l=l.querySelectorAll("input[name="+JSON.stringify(""+o)+'][type="radio"]'),o=0;o<l.length;o++){var d=l[o];if(d!==e&&d.form===e.form){var m=Ti(d);if(!m)throw Error(n(90));We(d),tt(d,m)}}}break;case"textarea":Oe(e,l);break;case"select":o=l.value,o!=null&&ye(e,!!l.multiple,o,!1)}},yn=xu,Kr=ho;var Ov={usingClientEntryPoint:!1,Events:[Sa,jo,Ti,eo,Do,xu]},Oa={findFiberByHostInstance:ao,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},Mv={bundleType:Oa.bundleType,version:Oa.version,rendererPackageName:Oa.rendererPackageName,rendererConfig:Oa.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:T.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Xe(e),e===null?null:e.stateNode},findFiberByHostInstance:Oa.findFiberByHostInstance||Bv,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var hs=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!hs.isDisabled&&hs.supportsFiber)try{Yr=hs.inject(Mv),Qt=hs}catch{}}return sr.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=Ov,sr.createPortal=function(e,o){var l=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!Au(o))throw Error(n(200));return Iv(e,o,null,l)},sr.createRoot=function(e,o){if(!Au(e))throw Error(n(299));var l=!1,d="",m=fp;return o!=null&&(o.unstable_strictMode===!0&&(l=!0),o.identifierPrefix!==void 0&&(d=o.identifierPrefix),o.onRecoverableError!==void 0&&(m=o.onRecoverableError)),o=bu(e,1,!1,null,null,l,!1,d,m),e[Zr]=o.current,Ea(e.nodeType===8?e.parentNode:e),new Ru(o)},sr.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var o=e._reactInternals;if(o===void 0)throw typeof e.render=="function"?Error(n(188)):(e=Object.keys(e).join(","),Error(n(268,e)));return e=Xe(o),e=e===null?null:e.stateNode,e},sr.flushSync=function(e){return ho(e)},sr.hydrate=function(e,o,l){if(!fs(o))throw Error(n(200));return ps(null,e,o,!0,l)},sr.hydrateRoot=function(e,o,l){if(!Au(e))throw Error(n(405));var d=l!=null&&l.hydratedSources||null,m=!1,v="",R=fp;if(l!=null&&(l.unstable_strictMode===!0&&(m=!0),l.identifierPrefix!==void 0&&(v=l.identifierPrefix),l.onRecoverableError!==void 0&&(R=l.onRecoverableError)),o=up(o,null,e,1,l??null,m,!1,v,R),e[Zr]=o.current,Ea(e),d)for(e=0;e<d.length;e++)l=d[e],m=l._getVersion,m=m(l._source),o.mutableSourceEagerHydrationData==null?o.mutableSourceEagerHydrationData=[l,m]:o.mutableSourceEagerHydrationData.push(l,m);return new ds(o)},sr.render=function(e,o,l){if(!fs(o))throw Error(n(200));return ps(null,e,o,!1,l)},sr.unmountComponentAtNode=function(e){if(!fs(e))throw Error(n(40));return e._reactRootContainer?(ho(function(){ps(null,null,e,!1,function(){e._reactRootContainer=null,e[Zr]=null})}),!0):!1},sr.unstable_batchedUpdates=xu,sr.unstable_renderSubtreeIntoContainer=function(e,o,l,d){if(!fs(l))throw Error(n(200));if(e==null||e._reactInternals===void 0)throw Error(n(38));return ps(e,o,l,!1,d)},sr.version="18.3.1-next-f1338f8080-20240426",sr}var Ep;function _m(){if(Ep)return Tu.exports;Ep=1;function r(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(r)}catch(t){console.error(t)}}return r(),Tu.exports=Qv(),Tu.exports}var Cp;function Jv(){if(Cp)return ms;Cp=1;var r=_m();return ms.createRoot=r.createRoot,ms.hydrateRoot=r.hydrateRoot,ms}var Zv=Jv();const $e=r=>typeof r=="string",$a=()=>{let r,t;const n=new Promise((a,i)=>{r=a,t=i});return n.resolve=r,n.reject=t,n},bp=r=>r==null?"":""+r,ey=(r,t,n)=>{r.forEach(a=>{t[a]&&(n[a]=t[a])})},ty=/###/g,Sp=r=>r&&r.indexOf("###")>-1?r.replace(ty,"."):r,Rp=r=>!r||$e(r),Qa=(r,t,n)=>{const a=$e(t)?t.split("."):t;let i=0;for(;i<a.length-1;){if(Rp(r))return{};const s=Sp(a[i]);!r[s]&&n&&(r[s]=new n),Object.prototype.hasOwnProperty.call(r,s)?r=r[s]:r={},++i}return Rp(r)?{}:{obj:r,k:Sp(a[i])}},Ap=(r,t,n)=>{const{obj:a,k:i}=Qa(r,t,Object);if(a!==void 0||t.length===1){a[i]=n;return}let s=t[t.length-1],c=t.slice(0,t.length-1),u=Qa(r,c,Object);for(;u.obj===void 0&&c.length;)s=`${c[c.length-1]}.${s}`,c=c.slice(0,c.length-1),u=Qa(r,c,Object),u!=null&&u.obj&&typeof u.obj[`${u.k}.${s}`]<"u"&&(u.obj=void 0);u.obj[`${u.k}.${s}`]=n},ry=(r,t,n,a)=>{const{obj:i,k:s}=Qa(r,t,Object);i[s]=i[s]||[],i[s].push(n)},El=(r,t)=>{const{obj:n,k:a}=Qa(r,t);if(n&&Object.prototype.hasOwnProperty.call(n,a))return n[a]},ny=(r,t,n)=>{const a=El(r,n);return a!==void 0?a:El(t,n)},Lm=(r,t,n)=>{for(const a in t)a!=="__proto__"&&a!=="constructor"&&(a in r?$e(r[a])||r[a]instanceof String||$e(t[a])||t[a]instanceof String?n&&(r[a]=t[a]):Lm(r[a],t[a],n):r[a]=t[a]);return r},Jo=r=>r.replace(/[\-\[\]\/\{\}\(\)\*\+\?\.\\\^\$\|]/g,"\\$&");var oy={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;","/":"&#x2F;"};const ay=r=>$e(r)?r.replace(/[&<>"'\/]/g,t=>oy[t]):r;class iy{constructor(t){this.capacity=t,this.regExpMap=new Map,this.regExpQueue=[]}getRegExp(t){const n=this.regExpMap.get(t);if(n!==void 0)return n;const a=new RegExp(t);return this.regExpQueue.length===this.capacity&&this.regExpMap.delete(this.regExpQueue.shift()),this.regExpMap.set(t,a),this.regExpQueue.push(t),a}}const sy=[" ",",","?","!",";"],ly=new iy(20),cy=(r,t,n)=>{t=t||"",n=n||"";const a=sy.filter(c=>t.indexOf(c)<0&&n.indexOf(c)<0);if(a.length===0)return!0;const i=ly.getRegExp(`(${a.map(c=>c==="?"?"\\?":c).join("|")})`);let s=!i.test(r);if(!s){const c=r.indexOf(n);c>0&&!i.test(r.substring(0,c))&&(s=!0)}return s},od=(r,t,n=".")=>{if(!r)return;if(r[t])return Object.prototype.hasOwnProperty.call(r,t)?r[t]:void 0;const a=t.split(n);let i=r;for(let s=0;s<a.length;){if(!i||typeof i!="object")return;let c,u="";for(let p=s;p<a.length;++p)if(p!==s&&(u+=n),u+=a[p],c=i[u],c!==void 0){if(["string","number","boolean"].indexOf(typeof c)>-1&&p<a.length-1)continue;s+=p-s+1;break}i=c}return i},ei=r=>r==null?void 0:r.replace("_","-"),uy={type:"logger",log(r){this.output("log",r)},warn(r){this.output("warn",r)},error(r){this.output("error",r)},output(r,t){var n,a;(a=(n=console==null?void 0:console[r])==null?void 0:n.apply)==null||a.call(n,console,t)}};class Cl{constructor(t,n={}){this.init(t,n)}init(t,n={}){this.prefix=n.prefix||"i18next:",this.logger=t||uy,this.options=n,this.debug=n.debug}log(...t){return this.forward(t,"log","",!0)}warn(...t){return this.forward(t,"warn","",!0)}error(...t){return this.forward(t,"error","")}deprecate(...t){return this.forward(t,"warn","WARNING DEPRECATED: ",!0)}forward(t,n,a,i){return i&&!this.debug?null:($e(t[0])&&(t[0]=`${a}${this.prefix} ${t[0]}`),this.logger[n](t))}create(t){return new Cl(this.logger,{prefix:`${this.prefix}:${t}:`,...this.options})}clone(t){return t=t||this.options,t.prefix=t.prefix||this.prefix,new Cl(this.logger,t)}}var Wr=new Cl;class Fl{constructor(){this.observers={}}on(t,n){return t.split(" ").forEach(a=>{this.observers[a]||(this.observers[a]=new Map);const i=this.observers[a].get(n)||0;this.observers[a].set(n,i+1)}),this}off(t,n){if(this.observers[t]){if(!n){delete this.observers[t];return}this.observers[t].delete(n)}}emit(t,...n){this.observers[t]&&Array.from(this.observers[t].entries()).forEach(([i,s])=>{for(let c=0;c<s;c++)i(...n)}),this.observers["*"]&&Array.from(this.observers["*"].entries()).forEach(([i,s])=>{for(let c=0;c<s;c++)i.apply(i,[t,...n])})}}class Pp extends Fl{constructor(t,n={ns:["translation"],defaultNS:"translation"}){super(),this.data=t||{},this.options=n,this.options.keySeparator===void 0&&(this.options.keySeparator="."),this.options.ignoreJSONStructure===void 0&&(this.options.ignoreJSONStructure=!0)}addNamespaces(t){this.options.ns.indexOf(t)<0&&this.options.ns.push(t)}removeNamespaces(t){const n=this.options.ns.indexOf(t);n>-1&&this.options.ns.splice(n,1)}getResource(t,n,a,i={}){var f,h;const s=i.keySeparator!==void 0?i.keySeparator:this.options.keySeparator,c=i.ignoreJSONStructure!==void 0?i.ignoreJSONStructure:this.options.ignoreJSONStructure;let u;t.indexOf(".")>-1?u=t.split("."):(u=[t,n],a&&(Array.isArray(a)?u.push(...a):$e(a)&&s?u.push(...a.split(s)):u.push(a)));const p=El(this.data,u);return!p&&!n&&!a&&t.indexOf(".")>-1&&(t=u[0],n=u[1],a=u.slice(2).join(".")),p||!c||!$e(a)?p:od((h=(f=this.data)==null?void 0:f[t])==null?void 0:h[n],a,s)}addResource(t,n,a,i,s={silent:!1}){const c=s.keySeparator!==void 0?s.keySeparator:this.options.keySeparator;let u=[t,n];a&&(u=u.concat(c?a.split(c):a)),t.indexOf(".")>-1&&(u=t.split("."),i=n,n=u[1]),this.addNamespaces(n),Ap(this.data,u,i),s.silent||this.emit("added",t,n,a,i)}addResources(t,n,a,i={silent:!1}){for(const s in a)($e(a[s])||Array.isArray(a[s]))&&this.addResource(t,n,s,a[s],{silent:!0});i.silent||this.emit("added",t,n,a)}addResourceBundle(t,n,a,i,s,c={silent:!1,skipCopy:!1}){let u=[t,n];t.indexOf(".")>-1&&(u=t.split("."),i=a,a=n,n=u[1]),this.addNamespaces(n);let p=El(this.data,u)||{};c.skipCopy||(a=JSON.parse(JSON.stringify(a))),i?Lm(p,a,s):p={...p,...a},Ap(this.data,u,p),c.silent||this.emit("added",t,n,a)}removeResourceBundle(t,n){this.hasResourceBundle(t,n)&&delete this.data[t][n],this.removeNamespaces(n),this.emit("removed",t,n)}hasResourceBundle(t,n){return this.getResource(t,n)!==void 0}getResourceBundle(t,n){return n||(n=this.options.defaultNS),this.getResource(t,n)}getDataByLanguage(t){return this.data[t]}hasLanguageSomeTranslations(t){const n=this.getDataByLanguage(t);return!!(n&&Object.keys(n)||[]).find(i=>n[i]&&Object.keys(n[i]).length>0)}toJSON(){return this.data}}var Fm={processors:{},addPostProcessor(r){this.processors[r.name]=r},handle(r,t,n,a,i){return r.forEach(s=>{var c;t=((c=this.processors[s])==null?void 0:c.process(t,n,a,i))??t}),t}};const Nm=Symbol("i18next/PATH_KEY");function dy(){const r=[],t=Object.create(null);let n;return t.get=(a,i)=>{var s;return(s=n==null?void 0:n.revoke)==null||s.call(n),i===Nm?r:(r.push(i),n=Proxy.revocable(a,t),n.proxy)},Proxy.revocable(Object.create(null),t).proxy}function ad(r,t){const{[Nm]:n}=r(dy());return n.join((t==null?void 0:t.keySeparator)??".")}const kp={},Fu=r=>!$e(r)&&typeof r!="boolean"&&typeof r!="number";class bl extends Fl{constructor(t,n={}){super(),ey(["resourceStore","languageUtils","pluralResolver","interpolator","backendConnector","i18nFormat","utils"],t,this),this.options=n,this.options.keySeparator===void 0&&(this.options.keySeparator="."),this.logger=Wr.create("translator")}changeLanguage(t){t&&(this.language=t)}exists(t,n={interpolation:{}}){const a={...n};if(t==null)return!1;const i=this.resolve(t,a);if((i==null?void 0:i.res)===void 0)return!1;const s=Fu(i.res);return!(a.returnObjects===!1&&s)}extractFromKey(t,n){let a=n.nsSeparator!==void 0?n.nsSeparator:this.options.nsSeparator;a===void 0&&(a=":");const i=n.keySeparator!==void 0?n.keySeparator:this.options.keySeparator;let s=n.ns||this.options.defaultNS||[];const c=a&&t.indexOf(a)>-1,u=!this.options.userDefinedKeySeparator&&!n.keySeparator&&!this.options.userDefinedNsSeparator&&!n.nsSeparator&&!cy(t,a,i);if(c&&!u){const p=t.match(this.interpolator.nestingRegexp);if(p&&p.length>0)return{key:t,namespaces:$e(s)?[s]:s};const f=t.split(a);(a!==i||a===i&&this.options.ns.indexOf(f[0])>-1)&&(s=f.shift()),t=f.join(i)}return{key:t,namespaces:$e(s)?[s]:s}}translate(t,n,a){let i=typeof n=="object"?{...n}:n;if(typeof i!="object"&&this.options.overloadTranslationOptionHandler&&(i=this.options.overloadTranslationOptionHandler(arguments)),typeof i=="object"&&(i={...i}),i||(i={}),t==null)return"";typeof t=="function"&&(t=ad(t,{...this.options,...i})),Array.isArray(t)||(t=[String(t)]);const s=i.returnDetails!==void 0?i.returnDetails:this.options.returnDetails,c=i.keySeparator!==void 0?i.keySeparator:this.options.keySeparator,{key:u,namespaces:p}=this.extractFromKey(t[t.length-1],i),f=p[p.length-1];let h=i.nsSeparator!==void 0?i.nsSeparator:this.options.nsSeparator;h===void 0&&(h=":");const x=i.lng||this.language,y=i.appendNamespaceToCIMode||this.options.appendNamespaceToCIMode;if((x==null?void 0:x.toLowerCase())==="cimode")return y?s?{res:`${f}${h}${u}`,usedKey:u,exactUsedKey:u,usedLng:x,usedNS:f,usedParams:this.getUsedParamsDetails(i)}:`${f}${h}${u}`:s?{res:u,usedKey:u,exactUsedKey:u,usedLng:x,usedNS:f,usedParams:this.getUsedParamsDetails(i)}:u;const w=this.resolve(t,i);let E=w==null?void 0:w.res;const C=(w==null?void 0:w.usedKey)||u,S=(w==null?void 0:w.exactUsedKey)||u,P=["[object Number]","[object Function]","[object RegExp]"],b=i.joinArrays!==void 0?i.joinArrays:this.options.joinArrays,A=!this.i18nFormat||this.i18nFormat.handleAsObject,T=i.count!==void 0&&!$e(i.count),B=bl.hasDefaultValue(i),F=T?this.pluralResolver.getSuffix(x,i.count,i):"",k=i.ordinal&&T?this.pluralResolver.getSuffix(x,i.count,{ordinal:!1}):"",N=T&&!i.ordinal&&i.count===0,I=N&&i[`defaultValue${this.options.pluralSeparator}zero`]||i[`defaultValue${F}`]||i[`defaultValue${k}`]||i.defaultValue;let L=E;A&&!E&&B&&(L=I);const M=Fu(L),H=Object.prototype.toString.apply(L);if(A&&L&&M&&P.indexOf(H)<0&&!($e(b)&&Array.isArray(L))){if(!i.returnObjects&&!this.options.returnObjects){this.options.returnedObjectHandler||this.logger.warn("accessing an object - but returnObjects options is not enabled!");const z=this.options.returnedObjectHandler?this.options.returnedObjectHandler(C,L,{...i,ns:p}):`key '${u} (${this.language})' returned an object instead of string.`;return s?(w.res=z,w.usedParams=this.getUsedParamsDetails(i),w):z}if(c){const z=Array.isArray(L),Z=z?[]:{},oe=z?S:C;for(const re in L)if(Object.prototype.hasOwnProperty.call(L,re)){const se=`${oe}${c}${re}`;B&&!E?Z[re]=this.translate(se,{...i,defaultValue:Fu(I)?I[re]:void 0,joinArrays:!1,ns:p}):Z[re]=this.translate(se,{...i,joinArrays:!1,ns:p}),Z[re]===se&&(Z[re]=L[re])}E=Z}}else if(A&&$e(b)&&Array.isArray(E))E=E.join(b),E&&(E=this.extendTranslation(E,t,i,a));else{let z=!1,Z=!1;!this.isValidLookup(E)&&B&&(z=!0,E=I),this.isValidLookup(E)||(Z=!0,E=u);const re=(i.missingKeyNoValueFallbackToKey||this.options.missingKeyNoValueFallbackToKey)&&Z?void 0:E,se=B&&I!==E&&this.options.updateMissing;if(Z||z||se){if(this.logger.log(se?"updateKey":"missingKey",x,f,u,se?I:E),c){const _=this.resolve(u,{...i,keySeparator:!1});_&&_.res&&this.logger.warn("Seems the loaded translations were in flat JSON format instead of nested. Either set keySeparator: false on init or make sure your translations are published in nested format.")}let Y=[];const te=this.languageUtils.getFallbackCodes(this.options.fallbackLng,i.lng||this.language);if(this.options.saveMissingTo==="fallback"&&te&&te[0])for(let _=0;_<te.length;_++)Y.push(te[_]);else this.options.saveMissingTo==="all"?Y=this.languageUtils.toResolveHierarchy(i.lng||this.language):Y.push(i.lng||this.language);const ee=(_,$,G)=>{var ue;const K=B&&G!==E?G:re;this.options.missingKeyHandler?this.options.missingKeyHandler(_,f,$,K,se,i):(ue=this.backendConnector)!=null&&ue.saveMissing&&this.backendConnector.saveMissing(_,f,$,K,se,i),this.emit("missingKey",_,f,$,E)};this.options.saveMissing&&(this.options.saveMissingPlurals&&T?Y.forEach(_=>{const $=this.pluralResolver.getSuffixes(_,i);N&&i[`defaultValue${this.options.pluralSeparator}zero`]&&$.indexOf(`${this.options.pluralSeparator}zero`)<0&&$.push(`${this.options.pluralSeparator}zero`),$.forEach(G=>{ee([_],u+G,i[`defaultValue${G}`]||I)})}):ee(Y,u,I))}E=this.extendTranslation(E,t,i,w,a),Z&&E===u&&this.options.appendNamespaceToMissingKey&&(E=`${f}${h}${u}`),(Z||z)&&this.options.parseMissingKeyHandler&&(E=this.options.parseMissingKeyHandler(this.options.appendNamespaceToMissingKey?`${f}${h}${u}`:u,z?E:void 0,i))}return s?(w.res=E,w.usedParams=this.getUsedParamsDetails(i),w):E}extendTranslation(t,n,a,i,s){var p,f;if((p=this.i18nFormat)!=null&&p.parse)t=this.i18nFormat.parse(t,{...this.options.interpolation.defaultVariables,...a},a.lng||this.language||i.usedLng,i.usedNS,i.usedKey,{resolved:i});else if(!a.skipInterpolation){a.interpolation&&this.interpolator.init({...a,interpolation:{...this.options.interpolation,...a.interpolation}});const h=$e(t)&&(((f=a==null?void 0:a.interpolation)==null?void 0:f.skipOnVariables)!==void 0?a.interpolation.skipOnVariables:this.options.interpolation.skipOnVariables);let x;if(h){const w=t.match(this.interpolator.nestingRegexp);x=w&&w.length}let y=a.replace&&!$e(a.replace)?a.replace:a;if(this.options.interpolation.defaultVariables&&(y={...this.options.interpolation.defaultVariables,...y}),t=this.interpolator.interpolate(t,y,a.lng||this.language||i.usedLng,a),h){const w=t.match(this.interpolator.nestingRegexp),E=w&&w.length;x<E&&(a.nest=!1)}!a.lng&&i&&i.res&&(a.lng=this.language||i.usedLng),a.nest!==!1&&(t=this.interpolator.nest(t,(...w)=>(s==null?void 0:s[0])===w[0]&&!a.context?(this.logger.warn(`It seems you are nesting recursively key: ${w[0]} in key: ${n[0]}`),null):this.translate(...w,n),a)),a.interpolation&&this.interpolator.reset()}const c=a.postProcess||this.options.postProcess,u=$e(c)?[c]:c;return t!=null&&(u!=null&&u.length)&&a.applyPostProcessor!==!1&&(t=Fm.handle(u,t,n,this.options&&this.options.postProcessPassResolved?{i18nResolved:{...i,usedParams:this.getUsedParamsDetails(a)},...a}:a,this)),t}resolve(t,n={}){let a,i,s,c,u;return $e(t)&&(t=[t]),t.forEach(p=>{if(this.isValidLookup(a))return;const f=this.extractFromKey(p,n),h=f.key;i=h;let x=f.namespaces;this.options.fallbackNS&&(x=x.concat(this.options.fallbackNS));const y=n.count!==void 0&&!$e(n.count),w=y&&!n.ordinal&&n.count===0,E=n.context!==void 0&&($e(n.context)||typeof n.context=="number")&&n.context!=="",C=n.lngs?n.lngs:this.languageUtils.toResolveHierarchy(n.lng||this.language,n.fallbackLng);x.forEach(S=>{var P,b;this.isValidLookup(a)||(u=S,!kp[`${C[0]}-${S}`]&&((P=this.utils)!=null&&P.hasLoadedNamespace)&&!((b=this.utils)!=null&&b.hasLoadedNamespace(u))&&(kp[`${C[0]}-${S}`]=!0,this.logger.warn(`key "${i}" for languages "${C.join(", ")}" won't get resolved as namespace "${u}" was not yet loaded`,"This means something IS WRONG in your setup. You access the t function before i18next.init / i18next.loadNamespace / i18next.changeLanguage was done. Wait for the callback or Promise to resolve before accessing it!!!")),C.forEach(A=>{var F;if(this.isValidLookup(a))return;c=A;const T=[h];if((F=this.i18nFormat)!=null&&F.addLookupKeys)this.i18nFormat.addLookupKeys(T,h,A,S,n);else{let k;y&&(k=this.pluralResolver.getSuffix(A,n.count,n));const N=`${this.options.pluralSeparator}zero`,I=`${this.options.pluralSeparator}ordinal${this.options.pluralSeparator}`;if(y&&(n.ordinal&&k.indexOf(I)===0&&T.push(h+k.replace(I,this.options.pluralSeparator)),T.push(h+k),w&&T.push(h+N)),E){const L=`${h}${this.options.contextSeparator||"_"}${n.context}`;T.push(L),y&&(n.ordinal&&k.indexOf(I)===0&&T.push(L+k.replace(I,this.options.pluralSeparator)),T.push(L+k),w&&T.push(L+N))}}let B;for(;B=T.pop();)this.isValidLookup(a)||(s=B,a=this.getResource(A,S,B,n))}))})}),{res:a,usedKey:i,exactUsedKey:s,usedLng:c,usedNS:u}}isValidLookup(t){return t!==void 0&&!(!this.options.returnNull&&t===null)&&!(!this.options.returnEmptyString&&t==="")}getResource(t,n,a,i={}){var s;return(s=this.i18nFormat)!=null&&s.getResource?this.i18nFormat.getResource(t,n,a,i):this.resourceStore.getResource(t,n,a,i)}getUsedParamsDetails(t={}){const n=["defaultValue","ordinal","context","replace","lng","lngs","fallbackLng","ns","keySeparator","nsSeparator","returnObjects","returnDetails","joinArrays","postProcess","interpolation"],a=t.replace&&!$e(t.replace);let i=a?t.replace:t;if(a&&typeof t.count<"u"&&(i.count=t.count),this.options.interpolation.defaultVariables&&(i={...this.options.interpolation.defaultVariables,...i}),!a){i={...i};for(const s of n)delete i[s]}return i}static hasDefaultValue(t){const n="defaultValue";for(const a in t)if(Object.prototype.hasOwnProperty.call(t,a)&&n===a.substring(0,n.length)&&t[a]!==void 0)return!0;return!1}}class Dp{constructor(t){this.options=t,this.supportedLngs=this.options.supportedLngs||!1,this.logger=Wr.create("languageUtils")}getScriptPartFromCode(t){if(t=ei(t),!t||t.indexOf("-")<0)return null;const n=t.split("-");return n.length===2||(n.pop(),n[n.length-1].toLowerCase()==="x")?null:this.formatLanguageCode(n.join("-"))}getLanguagePartFromCode(t){if(t=ei(t),!t||t.indexOf("-")<0)return t;const n=t.split("-");return this.formatLanguageCode(n[0])}formatLanguageCode(t){if($e(t)&&t.indexOf("-")>-1){let n;try{n=Intl.getCanonicalLocales(t)[0]}catch{}return n&&this.options.lowerCaseLng&&(n=n.toLowerCase()),n||(this.options.lowerCaseLng?t.toLowerCase():t)}return this.options.cleanCode||this.options.lowerCaseLng?t.toLowerCase():t}isSupportedCode(t){return(this.options.load==="languageOnly"||this.options.nonExplicitSupportedLngs)&&(t=this.getLanguagePartFromCode(t)),!this.supportedLngs||!this.supportedLngs.length||this.supportedLngs.indexOf(t)>-1}getBestMatchFromCodes(t){if(!t)return null;let n;return t.forEach(a=>{if(n)return;const i=this.formatLanguageCode(a);(!this.options.supportedLngs||this.isSupportedCode(i))&&(n=i)}),!n&&this.options.supportedLngs&&t.forEach(a=>{if(n)return;const i=this.getScriptPartFromCode(a);if(this.isSupportedCode(i))return n=i;const s=this.getLanguagePartFromCode(a);if(this.isSupportedCode(s))return n=s;n=this.options.supportedLngs.find(c=>{if(c===s)return c;if(!(c.indexOf("-")<0&&s.indexOf("-")<0)&&(c.indexOf("-")>0&&s.indexOf("-")<0&&c.substring(0,c.indexOf("-"))===s||c.indexOf(s)===0&&s.length>1))return c})}),n||(n=this.getFallbackCodes(this.options.fallbackLng)[0]),n}getFallbackCodes(t,n){if(!t)return[];if(typeof t=="function"&&(t=t(n)),$e(t)&&(t=[t]),Array.isArray(t))return t;if(!n)return t.default||[];let a=t[n];return a||(a=t[this.getScriptPartFromCode(n)]),a||(a=t[this.formatLanguageCode(n)]),a||(a=t[this.getLanguagePartFromCode(n)]),a||(a=t.default),a||[]}toResolveHierarchy(t,n){const a=this.getFallbackCodes((n===!1?[]:n)||this.options.fallbackLng||[],t),i=[],s=c=>{c&&(this.isSupportedCode(c)?i.push(c):this.logger.warn(`rejecting language code not found in supportedLngs: ${c}`))};return $e(t)&&(t.indexOf("-")>-1||t.indexOf("_")>-1)?(this.options.load!=="languageOnly"&&s(this.formatLanguageCode(t)),this.options.load!=="languageOnly"&&this.options.load!=="currentOnly"&&s(this.getScriptPartFromCode(t)),this.options.load!=="currentOnly"&&s(this.getLanguagePartFromCode(t))):$e(t)&&s(this.formatLanguageCode(t)),a.forEach(c=>{i.indexOf(c)<0&&s(this.formatLanguageCode(c))}),i}}const Tp={zero:0,one:1,two:2,few:3,many:4,other:5},_p={select:r=>r===1?"one":"other",resolvedOptions:()=>({pluralCategories:["one","other"]})};class fy{constructor(t,n={}){this.languageUtils=t,this.options=n,this.logger=Wr.create("pluralResolver"),this.pluralRulesCache={}}clearCache(){this.pluralRulesCache={}}getRule(t,n={}){const a=ei(t==="dev"?"en":t),i=n.ordinal?"ordinal":"cardinal",s=JSON.stringify({cleanedCode:a,type:i});if(s in this.pluralRulesCache)return this.pluralRulesCache[s];let c;try{c=new Intl.PluralRules(a,{type:i})}catch{if(!Intl)return this.logger.error("No Intl support, please use an Intl polyfill!"),_p;if(!t.match(/-|_/))return _p;const p=this.languageUtils.getLanguagePartFromCode(t);c=this.getRule(p,n)}return this.pluralRulesCache[s]=c,c}needsPlural(t,n={}){let a=this.getRule(t,n);return a||(a=this.getRule("dev",n)),(a==null?void 0:a.resolvedOptions().pluralCategories.length)>1}getPluralFormsOfKey(t,n,a={}){return this.getSuffixes(t,a).map(i=>`${n}${i}`)}getSuffixes(t,n={}){let a=this.getRule(t,n);return a||(a=this.getRule("dev",n)),a?a.resolvedOptions().pluralCategories.sort((i,s)=>Tp[i]-Tp[s]).map(i=>`${this.options.prepend}${n.ordinal?`ordinal${this.options.prepend}`:""}${i}`):[]}getSuffix(t,n,a={}){const i=this.getRule(t,a);return i?`${this.options.prepend}${a.ordinal?`ordinal${this.options.prepend}`:""}${i.select(n)}`:(this.logger.warn(`no plural rule found for: ${t}`),this.getSuffix("dev",n,a))}}const Lp=(r,t,n,a=".",i=!0)=>{let s=ny(r,t,n);return!s&&i&&$e(n)&&(s=od(r,n,a),s===void 0&&(s=od(t,n,a))),s},Nu=r=>r.replace(/\$/g,"$$$$");class Fp{constructor(t={}){var n;this.logger=Wr.create("interpolator"),this.options=t,this.format=((n=t==null?void 0:t.interpolation)==null?void 0:n.format)||(a=>a),this.init(t)}init(t={}){t.interpolation||(t.interpolation={escapeValue:!0});const{escape:n,escapeValue:a,useRawValueToEscape:i,prefix:s,prefixEscaped:c,suffix:u,suffixEscaped:p,formatSeparator:f,unescapeSuffix:h,unescapePrefix:x,nestingPrefix:y,nestingPrefixEscaped:w,nestingSuffix:E,nestingSuffixEscaped:C,nestingOptionsSeparator:S,maxReplaces:P,alwaysFormat:b}=t.interpolation;this.escape=n!==void 0?n:ay,this.escapeValue=a!==void 0?a:!0,this.useRawValueToEscape=i!==void 0?i:!1,this.prefix=s?Jo(s):c||"{{",this.suffix=u?Jo(u):p||"}}",this.formatSeparator=f||",",this.unescapePrefix=h?"":x||"-",this.unescapeSuffix=this.unescapePrefix?"":h||"",this.nestingPrefix=y?Jo(y):w||Jo("$t("),this.nestingSuffix=E?Jo(E):C||Jo(")"),this.nestingOptionsSeparator=S||",",this.maxReplaces=P||1e3,this.alwaysFormat=b!==void 0?b:!1,this.resetRegExp()}reset(){this.options&&this.init(this.options)}resetRegExp(){const t=(n,a)=>(n==null?void 0:n.source)===a?(n.lastIndex=0,n):new RegExp(a,"g");this.regexp=t(this.regexp,`${this.prefix}(.+?)${this.suffix}`),this.regexpUnescape=t(this.regexpUnescape,`${this.prefix}${this.unescapePrefix}(.+?)${this.unescapeSuffix}${this.suffix}`),this.nestingRegexp=t(this.nestingRegexp,`${this.nestingPrefix}((?:[^()"']+|"[^"]*"|'[^']*'|\\((?:[^()]|"[^"]*"|'[^']*')*\\))*?)${this.nestingSuffix}`)}interpolate(t,n,a,i){var w;let s,c,u;const p=this.options&&this.options.interpolation&&this.options.interpolation.defaultVariables||{},f=E=>{if(E.indexOf(this.formatSeparator)<0){const b=Lp(n,p,E,this.options.keySeparator,this.options.ignoreJSONStructure);return this.alwaysFormat?this.format(b,void 0,a,{...i,...n,interpolationkey:E}):b}const C=E.split(this.formatSeparator),S=C.shift().trim(),P=C.join(this.formatSeparator).trim();return this.format(Lp(n,p,S,this.options.keySeparator,this.options.ignoreJSONStructure),P,a,{...i,...n,interpolationkey:S})};this.resetRegExp();const h=(i==null?void 0:i.missingInterpolationHandler)||this.options.missingInterpolationHandler,x=((w=i==null?void 0:i.interpolation)==null?void 0:w.skipOnVariables)!==void 0?i.interpolation.skipOnVariables:this.options.interpolation.skipOnVariables;return[{regex:this.regexpUnescape,safeValue:E=>Nu(E)},{regex:this.regexp,safeValue:E=>this.escapeValue?Nu(this.escape(E)):Nu(E)}].forEach(E=>{for(u=0;s=E.regex.exec(t);){const C=s[1].trim();if(c=f(C),c===void 0)if(typeof h=="function"){const P=h(t,s,i);c=$e(P)?P:""}else if(i&&Object.prototype.hasOwnProperty.call(i,C))c="";else if(x){c=s[0];continue}else this.logger.warn(`missed to pass in variable ${C} for interpolating ${t}`),c="";else!$e(c)&&!this.useRawValueToEscape&&(c=bp(c));const S=E.safeValue(c);if(t=t.replace(s[0],S),x?(E.regex.lastIndex+=c.length,E.regex.lastIndex-=s[0].length):E.regex.lastIndex=0,u++,u>=this.maxReplaces)break}}),t}nest(t,n,a={}){let i,s,c;const u=(p,f)=>{const h=this.nestingOptionsSeparator;if(p.indexOf(h)<0)return p;const x=p.split(new RegExp(`${h}[ ]*{`));let y=`{${x[1]}`;p=x[0],y=this.interpolate(y,c);const w=y.match(/'/g),E=y.match(/"/g);(((w==null?void 0:w.length)??0)%2===0&&!E||E.length%2!==0)&&(y=y.replace(/'/g,'"'));try{c=JSON.parse(y),f&&(c={...f,...c})}catch(C){return this.logger.warn(`failed parsing options string in nesting for key ${p}`,C),`${p}${h}${y}`}return c.defaultValue&&c.defaultValue.indexOf(this.prefix)>-1&&delete c.defaultValue,p};for(;i=this.nestingRegexp.exec(t);){let p=[];c={...a},c=c.replace&&!$e(c.replace)?c.replace:c,c.applyPostProcessor=!1,delete c.defaultValue;const f=/{.*}/.test(i[1])?i[1].lastIndexOf("}")+1:i[1].indexOf(this.formatSeparator);if(f!==-1&&(p=i[1].slice(f).split(this.formatSeparator).map(h=>h.trim()).filter(Boolean),i[1]=i[1].slice(0,f)),s=n(u.call(this,i[1].trim(),c),c),s&&i[0]===t&&!$e(s))return s;$e(s)||(s=bp(s)),s||(this.logger.warn(`missed to resolve ${i[1]} for nesting ${t}`),s=""),p.length&&(s=p.reduce((h,x)=>this.format(h,x,a.lng,{...a,interpolationkey:i[1].trim()}),s.trim())),t=t.replace(i[0],s),this.regexp.lastIndex=0}return t}}const py=r=>{let t=r.toLowerCase().trim();const n={};if(r.indexOf("(")>-1){const a=r.split("(");t=a[0].toLowerCase().trim();const i=a[1].substring(0,a[1].length-1);t==="currency"&&i.indexOf(":")<0?n.currency||(n.currency=i.trim()):t==="relativetime"&&i.indexOf(":")<0?n.range||(n.range=i.trim()):i.split(";").forEach(c=>{if(c){const[u,...p]=c.split(":"),f=p.join(":").trim().replace(/^'+|'+$/g,""),h=u.trim();n[h]||(n[h]=f),f==="false"&&(n[h]=!1),f==="true"&&(n[h]=!0),isNaN(f)||(n[h]=parseInt(f,10))}})}return{formatName:t,formatOptions:n}},Np=r=>{const t={};return(n,a,i)=>{let s=i;i&&i.interpolationkey&&i.formatParams&&i.formatParams[i.interpolationkey]&&i[i.interpolationkey]&&(s={...s,[i.interpolationkey]:void 0});const c=a+JSON.stringify(s);let u=t[c];return u||(u=r(ei(a),i),t[c]=u),u(n)}},hy=r=>(t,n,a)=>r(ei(n),a)(t);class my{constructor(t={}){this.logger=Wr.create("formatter"),this.options=t,this.init(t)}init(t,n={interpolation:{}}){this.formatSeparator=n.interpolation.formatSeparator||",";const a=n.cacheInBuiltFormats?Np:hy;this.formats={number:a((i,s)=>{const c=new Intl.NumberFormat(i,{...s});return u=>c.format(u)}),currency:a((i,s)=>{const c=new Intl.NumberFormat(i,{...s,style:"currency"});return u=>c.format(u)}),datetime:a((i,s)=>{const c=new Intl.DateTimeFormat(i,{...s});return u=>c.format(u)}),relativetime:a((i,s)=>{const c=new Intl.RelativeTimeFormat(i,{...s});return u=>c.format(u,s.range||"day")}),list:a((i,s)=>{const c=new Intl.ListFormat(i,{...s});return u=>c.format(u)})}}add(t,n){this.formats[t.toLowerCase().trim()]=n}addCached(t,n){this.formats[t.toLowerCase().trim()]=Np(n)}format(t,n,a,i={}){const s=n.split(this.formatSeparator);if(s.length>1&&s[0].indexOf("(")>1&&s[0].indexOf(")")<0&&s.find(u=>u.indexOf(")")>-1)){const u=s.findIndex(p=>p.indexOf(")")>-1);s[0]=[s[0],...s.splice(1,u)].join(this.formatSeparator)}return s.reduce((u,p)=>{var x;const{formatName:f,formatOptions:h}=py(p);if(this.formats[f]){let y=u;try{const w=((x=i==null?void 0:i.formatParams)==null?void 0:x[i.interpolationkey])||{},E=w.locale||w.lng||i.locale||i.lng||a;y=this.formats[f](u,E,{...h,...i,...w})}catch(w){this.logger.warn(w)}return y}else this.logger.warn(`there was no format function for ${f}`);return u},t)}}const gy=(r,t)=>{r.pending[t]!==void 0&&(delete r.pending[t],r.pendingCount--)};class xy extends Fl{constructor(t,n,a,i={}){var s,c;super(),this.backend=t,this.store=n,this.services=a,this.languageUtils=a.languageUtils,this.options=i,this.logger=Wr.create("backendConnector"),this.waitingReads=[],this.maxParallelReads=i.maxParallelReads||10,this.readingCalls=0,this.maxRetries=i.maxRetries>=0?i.maxRetries:5,this.retryTimeout=i.retryTimeout>=1?i.retryTimeout:350,this.state={},this.queue=[],(c=(s=this.backend)==null?void 0:s.init)==null||c.call(s,a,i.backend,i)}queueLoad(t,n,a,i){const s={},c={},u={},p={};return t.forEach(f=>{let h=!0;n.forEach(x=>{const y=`${f}|${x}`;!a.reload&&this.store.hasResourceBundle(f,x)?this.state[y]=2:this.state[y]<0||(this.state[y]===1?c[y]===void 0&&(c[y]=!0):(this.state[y]=1,h=!1,c[y]===void 0&&(c[y]=!0),s[y]===void 0&&(s[y]=!0),p[x]===void 0&&(p[x]=!0)))}),h||(u[f]=!0)}),(Object.keys(s).length||Object.keys(c).length)&&this.queue.push({pending:c,pendingCount:Object.keys(c).length,loaded:{},errors:[],callback:i}),{toLoad:Object.keys(s),pending:Object.keys(c),toLoadLanguages:Object.keys(u),toLoadNamespaces:Object.keys(p)}}loaded(t,n,a){const i=t.split("|"),s=i[0],c=i[1];n&&this.emit("failedLoading",s,c,n),!n&&a&&this.store.addResourceBundle(s,c,a,void 0,void 0,{skipCopy:!0}),this.state[t]=n?-1:2,n&&a&&(this.state[t]=0);const u={};this.queue.forEach(p=>{ry(p.loaded,[s],c),gy(p,t),n&&p.errors.push(n),p.pendingCount===0&&!p.done&&(Object.keys(p.loaded).forEach(f=>{u[f]||(u[f]={});const h=p.loaded[f];h.length&&h.forEach(x=>{u[f][x]===void 0&&(u[f][x]=!0)})}),p.done=!0,p.errors.length?p.callback(p.errors):p.callback())}),this.emit("loaded",u),this.queue=this.queue.filter(p=>!p.done)}read(t,n,a,i=0,s=this.retryTimeout,c){if(!t.length)return c(null,{});if(this.readingCalls>=this.maxParallelReads){this.waitingReads.push({lng:t,ns:n,fcName:a,tried:i,wait:s,callback:c});return}this.readingCalls++;const u=(f,h)=>{if(this.readingCalls--,this.waitingReads.length>0){const x=this.waitingReads.shift();this.read(x.lng,x.ns,x.fcName,x.tried,x.wait,x.callback)}if(f&&h&&i<this.maxRetries){setTimeout(()=>{this.read.call(this,t,n,a,i+1,s*2,c)},s);return}c(f,h)},p=this.backend[a].bind(this.backend);if(p.length===2){try{const f=p(t,n);f&&typeof f.then=="function"?f.then(h=>u(null,h)).catch(u):u(null,f)}catch(f){u(f)}return}return p(t,n,u)}prepareLoading(t,n,a={},i){if(!this.backend)return this.logger.warn("No backend was added via i18next.use. Will not load resources."),i&&i();$e(t)&&(t=this.languageUtils.toResolveHierarchy(t)),$e(n)&&(n=[n]);const s=this.queueLoad(t,n,a,i);if(!s.toLoad.length)return s.pending.length||i(),null;s.toLoad.forEach(c=>{this.loadOne(c)})}load(t,n,a){this.prepareLoading(t,n,{},a)}reload(t,n,a){this.prepareLoading(t,n,{reload:!0},a)}loadOne(t,n=""){const a=t.split("|"),i=a[0],s=a[1];this.read(i,s,"read",void 0,void 0,(c,u)=>{c&&this.logger.warn(`${n}loading namespace ${s} for language ${i} failed`,c),!c&&u&&this.logger.log(`${n}loaded namespace ${s} for language ${i}`,u),this.loaded(t,c,u)})}saveMissing(t,n,a,i,s,c={},u=()=>{}){var p,f,h,x,y;if((f=(p=this.services)==null?void 0:p.utils)!=null&&f.hasLoadedNamespace&&!((x=(h=this.services)==null?void 0:h.utils)!=null&&x.hasLoadedNamespace(n))){this.logger.warn(`did not save key "${a}" as the namespace "${n}" was not yet loaded`,"This means something IS WRONG in your setup. You access the t function before i18next.init / i18next.loadNamespace / i18next.changeLanguage was done. Wait for the callback or Promise to resolve before accessing it!!!");return}if(!(a==null||a==="")){if((y=this.backend)!=null&&y.create){const w={...c,isUpdate:s},E=this.backend.create.bind(this.backend);if(E.length<6)try{let C;E.length===5?C=E(t,n,a,i,w):C=E(t,n,a,i),C&&typeof C.then=="function"?C.then(S=>u(null,S)).catch(u):u(null,C)}catch(C){u(C)}else E(t,n,a,i,u,w)}!t||!t[0]||this.store.addResource(t[0],n,a,i)}}}const Iu=()=>({debug:!1,initAsync:!0,ns:["translation"],defaultNS:["translation"],fallbackLng:["dev"],fallbackNS:!1,supportedLngs:!1,nonExplicitSupportedLngs:!1,load:"all",preload:!1,simplifyPluralSuffix:!0,keySeparator:".",nsSeparator:":",pluralSeparator:"_",contextSeparator:"_",partialBundledLanguages:!1,saveMissing:!1,updateMissing:!1,saveMissingTo:"fallback",saveMissingPlurals:!0,missingKeyHandler:!1,missingInterpolationHandler:!1,postProcess:!1,postProcessPassResolved:!1,returnNull:!1,returnEmptyString:!0,returnObjects:!1,joinArrays:!1,returnedObjectHandler:!1,parseMissingKeyHandler:!1,appendNamespaceToMissingKey:!1,appendNamespaceToCIMode:!1,overloadTranslationOptionHandler:r=>{let t={};if(typeof r[1]=="object"&&(t=r[1]),$e(r[1])&&(t.defaultValue=r[1]),$e(r[2])&&(t.tDescription=r[2]),typeof r[2]=="object"||typeof r[3]=="object"){const n=r[3]||r[2];Object.keys(n).forEach(a=>{t[a]=n[a]})}return t},interpolation:{escapeValue:!0,format:r=>r,prefix:"{{",suffix:"}}",formatSeparator:",",unescapePrefix:"-",nestingPrefix:"$t(",nestingSuffix:")",nestingOptionsSeparator:",",maxReplaces:1e3,skipOnVariables:!0},cacheInBuiltFormats:!0}),Ip=r=>{var t,n;return $e(r.ns)&&(r.ns=[r.ns]),$e(r.fallbackLng)&&(r.fallbackLng=[r.fallbackLng]),$e(r.fallbackNS)&&(r.fallbackNS=[r.fallbackNS]),((n=(t=r.supportedLngs)==null?void 0:t.indexOf)==null?void 0:n.call(t,"cimode"))<0&&(r.supportedLngs=r.supportedLngs.concat(["cimode"])),typeof r.initImmediate=="boolean"&&(r.initAsync=r.initImmediate),r},gs=()=>{},vy=r=>{Object.getOwnPropertyNames(Object.getPrototypeOf(r)).forEach(n=>{typeof r[n]=="function"&&(r[n]=r[n].bind(r))})},yy=r=>{var t,n,a,i,s,c,u,p,f;return!!(((a=(n=(t=r==null?void 0:r.modules)==null?void 0:t.backend)==null?void 0:n.name)==null?void 0:a.indexOf("Locize"))>0||((u=(c=(s=(i=r==null?void 0:r.modules)==null?void 0:i.backend)==null?void 0:s.constructor)==null?void 0:c.name)==null?void 0:u.indexOf("Locize"))>0||(f=(p=r==null?void 0:r.options)==null?void 0:p.backend)!=null&&f.backends&&r.options.backend.backends.some(h=>{var x,y,w;return((x=h==null?void 0:h.name)==null?void 0:x.indexOf("Locize"))>0||((w=(y=h==null?void 0:h.constructor)==null?void 0:y.name)==null?void 0:w.indexOf("Locize"))>0}))};class Ja extends Fl{constructor(t={},n){if(super(),this.options=Ip(t),this.services={},this.logger=Wr,this.modules={external:[]},vy(this),n&&!this.isInitialized&&!t.isClone){if(!this.options.initAsync)return this.init(t,n),this;setTimeout(()=>{this.init(t,n)},0)}}init(t={},n){this.isInitializing=!0,typeof t=="function"&&(n=t,t={}),t.defaultNS==null&&t.ns&&($e(t.ns)?t.defaultNS=t.ns:t.ns.indexOf("translation")<0&&(t.defaultNS=t.ns[0]));const a=Iu();this.options={...a,...this.options,...Ip(t)},this.options.interpolation={...a.interpolation,...this.options.interpolation},t.keySeparator!==void 0&&(this.options.userDefinedKeySeparator=t.keySeparator),t.nsSeparator!==void 0&&(this.options.userDefinedNsSeparator=t.nsSeparator),typeof this.options.overloadTranslationOptionHandler!="function"&&(this.options.overloadTranslationOptionHandler=a.overloadTranslationOptionHandler),this.options.showSupportNotice!==!1&&!yy(this)&&typeof console<"u"&&typeof console.info<"u"&&console.info("🌐 i18next is maintained with support from locize.com — consider powering your project with managed localization (AI, CDN, integrations): https://locize.com 💙");const i=f=>f?typeof f=="function"?new f:f:null;if(!this.options.isClone){this.modules.logger?Wr.init(i(this.modules.logger),this.options):Wr.init(null,this.options);let f;this.modules.formatter?f=this.modules.formatter:f=my;const h=new Dp(this.options);this.store=new Pp(this.options.resources,this.options);const x=this.services;x.logger=Wr,x.resourceStore=this.store,x.languageUtils=h,x.pluralResolver=new fy(h,{prepend:this.options.pluralSeparator,simplifyPluralSuffix:this.options.simplifyPluralSuffix}),this.options.interpolation.format&&this.options.interpolation.format!==a.interpolation.format&&this.logger.deprecate("init: you are still using the legacy format function, please use the new approach: https://www.i18next.com/translation-function/formatting"),f&&(!this.options.interpolation.format||this.options.interpolation.format===a.interpolation.format)&&(x.formatter=i(f),x.formatter.init&&x.formatter.init(x,this.options),this.options.interpolation.format=x.formatter.format.bind(x.formatter)),x.interpolator=new Fp(this.options),x.utils={hasLoadedNamespace:this.hasLoadedNamespace.bind(this)},x.backendConnector=new xy(i(this.modules.backend),x.resourceStore,x,this.options),x.backendConnector.on("*",(w,...E)=>{this.emit(w,...E)}),this.modules.languageDetector&&(x.languageDetector=i(this.modules.languageDetector),x.languageDetector.init&&x.languageDetector.init(x,this.options.detection,this.options)),this.modules.i18nFormat&&(x.i18nFormat=i(this.modules.i18nFormat),x.i18nFormat.init&&x.i18nFormat.init(this)),this.translator=new bl(this.services,this.options),this.translator.on("*",(w,...E)=>{this.emit(w,...E)}),this.modules.external.forEach(w=>{w.init&&w.init(this)})}if(this.format=this.options.interpolation.format,n||(n=gs),this.options.fallbackLng&&!this.services.languageDetector&&!this.options.lng){const f=this.services.languageUtils.getFallbackCodes(this.options.fallbackLng);f.length>0&&f[0]!=="dev"&&(this.options.lng=f[0])}!this.services.languageDetector&&!this.options.lng&&this.logger.warn("init: no languageDetector is used and no lng is defined"),["getResource","hasResourceBundle","getResourceBundle","getDataByLanguage"].forEach(f=>{this[f]=(...h)=>this.store[f](...h)}),["addResource","addResources","addResourceBundle","removeResourceBundle"].forEach(f=>{this[f]=(...h)=>(this.store[f](...h),this)});const u=$a(),p=()=>{const f=(h,x)=>{this.isInitializing=!1,this.isInitialized&&!this.initializedStoreOnce&&this.logger.warn("init: i18next is already initialized. You should call init just once!"),this.isInitialized=!0,this.options.isClone||this.logger.log("initialized",this.options),this.emit("initialized",this.options),u.resolve(x),n(h,x)};if(this.languages&&!this.isInitialized)return f(null,this.t.bind(this));this.changeLanguage(this.options.lng,f)};return this.options.resources||!this.options.initAsync?p():setTimeout(p,0),u}loadResources(t,n=gs){var s,c;let a=n;const i=$e(t)?t:this.language;if(typeof t=="function"&&(a=t),!this.options.resources||this.options.partialBundledLanguages){if((i==null?void 0:i.toLowerCase())==="cimode"&&(!this.options.preload||this.options.preload.length===0))return a();const u=[],p=f=>{if(!f||f==="cimode")return;this.services.languageUtils.toResolveHierarchy(f).forEach(x=>{x!=="cimode"&&u.indexOf(x)<0&&u.push(x)})};i?p(i):this.services.languageUtils.getFallbackCodes(this.options.fallbackLng).forEach(h=>p(h)),(c=(s=this.options.preload)==null?void 0:s.forEach)==null||c.call(s,f=>p(f)),this.services.backendConnector.load(u,this.options.ns,f=>{!f&&!this.resolvedLanguage&&this.language&&this.setResolvedLanguage(this.language),a(f)})}else a(null)}reloadResources(t,n,a){const i=$a();return typeof t=="function"&&(a=t,t=void 0),typeof n=="function"&&(a=n,n=void 0),t||(t=this.languages),n||(n=this.options.ns),a||(a=gs),this.services.backendConnector.reload(t,n,s=>{i.resolve(),a(s)}),i}use(t){if(!t)throw new Error("You are passing an undefined module! Please check the object you are passing to i18next.use()");if(!t.type)throw new Error("You are passing a wrong module! Please check the object you are passing to i18next.use()");return t.type==="backend"&&(this.modules.backend=t),(t.type==="logger"||t.log&&t.warn&&t.error)&&(this.modules.logger=t),t.type==="languageDetector"&&(this.modules.languageDetector=t),t.type==="i18nFormat"&&(this.modules.i18nFormat=t),t.type==="postProcessor"&&Fm.addPostProcessor(t),t.type==="formatter"&&(this.modules.formatter=t),t.type==="3rdParty"&&this.modules.external.push(t),this}setResolvedLanguage(t){if(!(!t||!this.languages)&&!(["cimode","dev"].indexOf(t)>-1)){for(let n=0;n<this.languages.length;n++){const a=this.languages[n];if(!(["cimode","dev"].indexOf(a)>-1)&&this.store.hasLanguageSomeTranslations(a)){this.resolvedLanguage=a;break}}!this.resolvedLanguage&&this.languages.indexOf(t)<0&&this.store.hasLanguageSomeTranslations(t)&&(this.resolvedLanguage=t,this.languages.unshift(t))}}changeLanguage(t,n){this.isLanguageChangingTo=t;const a=$a();this.emit("languageChanging",t);const i=u=>{this.language=u,this.languages=this.services.languageUtils.toResolveHierarchy(u),this.resolvedLanguage=void 0,this.setResolvedLanguage(u)},s=(u,p)=>{p?this.isLanguageChangingTo===t&&(i(p),this.translator.changeLanguage(p),this.isLanguageChangingTo=void 0,this.emit("languageChanged",p),this.logger.log("languageChanged",p)):this.isLanguageChangingTo=void 0,a.resolve((...f)=>this.t(...f)),n&&n(u,(...f)=>this.t(...f))},c=u=>{var h,x;!t&&!u&&this.services.languageDetector&&(u=[]);const p=$e(u)?u:u&&u[0],f=this.store.hasLanguageSomeTranslations(p)?p:this.services.languageUtils.getBestMatchFromCodes($e(u)?[u]:u);f&&(this.language||i(f),this.translator.language||this.translator.changeLanguage(f),(x=(h=this.services.languageDetector)==null?void 0:h.cacheUserLanguage)==null||x.call(h,f)),this.loadResources(f,y=>{s(y,f)})};return!t&&this.services.languageDetector&&!this.services.languageDetector.async?c(this.services.languageDetector.detect()):!t&&this.services.languageDetector&&this.services.languageDetector.async?this.services.languageDetector.detect.length===0?this.services.languageDetector.detect().then(c):this.services.languageDetector.detect(c):c(t),a}getFixedT(t,n,a){const i=(s,c,...u)=>{let p;typeof c!="object"?p=this.options.overloadTranslationOptionHandler([s,c].concat(u)):p={...c},p.lng=p.lng||i.lng,p.lngs=p.lngs||i.lngs,p.ns=p.ns||i.ns,p.keyPrefix!==""&&(p.keyPrefix=p.keyPrefix||a||i.keyPrefix);const f=this.options.keySeparator||".";let h;return p.keyPrefix&&Array.isArray(s)?h=s.map(x=>(typeof x=="function"&&(x=ad(x,{...this.options,...c})),`${p.keyPrefix}${f}${x}`)):(typeof s=="function"&&(s=ad(s,{...this.options,...c})),h=p.keyPrefix?`${p.keyPrefix}${f}${s}`:s),this.t(h,p)};return $e(t)?i.lng=t:i.lngs=t,i.ns=n,i.keyPrefix=a,i}t(...t){var n;return(n=this.translator)==null?void 0:n.translate(...t)}exists(...t){var n;return(n=this.translator)==null?void 0:n.exists(...t)}setDefaultNamespace(t){this.options.defaultNS=t}hasLoadedNamespace(t,n={}){if(!this.isInitialized)return this.logger.warn("hasLoadedNamespace: i18next was not initialized",this.languages),!1;if(!this.languages||!this.languages.length)return this.logger.warn("hasLoadedNamespace: i18n.languages were undefined or empty",this.languages),!1;const a=n.lng||this.resolvedLanguage||this.languages[0],i=this.options?this.options.fallbackLng:!1,s=this.languages[this.languages.length-1];if(a.toLowerCase()==="cimode")return!0;const c=(u,p)=>{const f=this.services.backendConnector.state[`${u}|${p}`];return f===-1||f===0||f===2};if(n.precheck){const u=n.precheck(this,c);if(u!==void 0)return u}return!!(this.hasResourceBundle(a,t)||!this.services.backendConnector.backend||this.options.resources&&!this.options.partialBundledLanguages||c(a,t)&&(!i||c(s,t)))}loadNamespaces(t,n){const a=$a();return this.options.ns?($e(t)&&(t=[t]),t.forEach(i=>{this.options.ns.indexOf(i)<0&&this.options.ns.push(i)}),this.loadResources(i=>{a.resolve(),n&&n(i)}),a):(n&&n(),Promise.resolve())}loadLanguages(t,n){const a=$a();$e(t)&&(t=[t]);const i=this.options.preload||[],s=t.filter(c=>i.indexOf(c)<0&&this.services.languageUtils.isSupportedCode(c));return s.length?(this.options.preload=i.concat(s),this.loadResources(c=>{a.resolve(),n&&n(c)}),a):(n&&n(),Promise.resolve())}dir(t){var i,s;if(t||(t=this.resolvedLanguage||(((i=this.languages)==null?void 0:i.length)>0?this.languages[0]:this.language)),!t)return"rtl";try{const c=new Intl.Locale(t);if(c&&c.getTextInfo){const u=c.getTextInfo();if(u&&u.direction)return u.direction}}catch{}const n=["ar","shu","sqr","ssh","xaa","yhd","yud","aao","abh","abv","acm","acq","acw","acx","acy","adf","ads","aeb","aec","afb","ajp","apc","apd","arb","arq","ars","ary","arz","auz","avl","ayh","ayl","ayn","ayp","bbz","pga","he","iw","ps","pbt","pbu","pst","prp","prd","ug","ur","ydd","yds","yih","ji","yi","hbo","men","xmn","fa","jpr","peo","pes","prs","dv","sam","ckb"],a=((s=this.services)==null?void 0:s.languageUtils)||new Dp(Iu());return t.toLowerCase().indexOf("-latn")>1?"ltr":n.indexOf(a.getLanguagePartFromCode(t))>-1||t.toLowerCase().indexOf("-arab")>1?"rtl":"ltr"}static createInstance(t={},n){const a=new Ja(t,n);return a.createInstance=Ja.createInstance,a}cloneInstance(t={},n=gs){const a=t.forkResourceStore;a&&delete t.forkResourceStore;const i={...this.options,...t,isClone:!0},s=new Ja(i);if((t.debug!==void 0||t.prefix!==void 0)&&(s.logger=s.logger.clone(t)),["store","services","language"].forEach(u=>{s[u]=this[u]}),s.services={...this.services},s.services.utils={hasLoadedNamespace:s.hasLoadedNamespace.bind(s)},a){const u=Object.keys(this.store.data).reduce((p,f)=>(p[f]={...this.store.data[f]},p[f]=Object.keys(p[f]).reduce((h,x)=>(h[x]={...p[f][x]},h),p[f]),p),{});s.store=new Pp(u,i),s.services.resourceStore=s.store}if(t.interpolation){const p={...Iu().interpolation,...this.options.interpolation,...t.interpolation},f={...i,interpolation:p};s.services.interpolator=new Fp(f)}return s.translator=new bl(s.services,i),s.translator.on("*",(u,...p)=>{s.emit(u,...p)}),s.init(i,n),s.translator.options=i,s.translator.backendConnector.services.utils={hasLoadedNamespace:s.hasLoadedNamespace.bind(s)},s}toJSON(){return{options:this.options,store:this.store,language:this.language,languages:this.languages,resolvedLanguage:this.resolvedLanguage}}}const Ct=Ja.createInstance();Ct.createInstance;Ct.dir;Ct.init;Ct.loadResources;Ct.reloadResources;Ct.use;Ct.changeLanguage;Ct.getFixedT;Ct.t;Ct.exists;Ct.setDefaultNamespace;Ct.hasLoadedNamespace;Ct.loadNamespaces;Ct.loadLanguages;const wy=(r,t,n,a)=>{var s,c,u,p;const i=[n,{code:t,...a||{}}];if((c=(s=r==null?void 0:r.services)==null?void 0:s.logger)!=null&&c.forward)return r.services.logger.forward(i,"warn","react-i18next::",!0);bo(i[0])&&(i[0]=`react-i18next:: ${i[0]}`),(p=(u=r==null?void 0:r.services)==null?void 0:u.logger)!=null&&p.warn?r.services.logger.warn(...i):console!=null&&console.warn&&console.warn(...i)},Bp={},Im=(r,t,n,a)=>{bo(n)&&Bp[n]||(bo(n)&&(Bp[n]=new Date),wy(r,t,n,a))},Bm=(r,t)=>()=>{if(r.isInitialized)t();else{const n=()=>{setTimeout(()=>{r.off("initialized",n)},0),t()};r.on("initialized",n)}},id=(r,t,n)=>{r.loadNamespaces(t,Bm(r,n))},jp=(r,t,n,a)=>{if(bo(n)&&(n=[n]),r.options.preload&&r.options.preload.indexOf(t)>-1)return id(r,n,a);n.forEach(i=>{r.options.ns.indexOf(i)<0&&r.options.ns.push(i)}),r.loadLanguages(t,Bm(r,a))},Ey=(r,t,n={})=>!t.languages||!t.languages.length?(Im(t,"NO_LANGUAGES","i18n.languages were undefined or empty",{languages:t.languages}),!0):t.hasLoadedNamespace(r,{lng:n.lng,precheck:(a,i)=>{if(n.bindI18n&&n.bindI18n.indexOf("languageChanging")>-1&&a.services.backendConnector.backend&&a.isLanguageChangingTo&&!i(a.isLanguageChangingTo,r))return!1}}),bo=r=>typeof r=="string",Cy=r=>typeof r=="object"&&r!==null,by=/&(?:amp|#38|lt|#60|gt|#62|apos|#39|quot|#34|nbsp|#160|copy|#169|reg|#174|hellip|#8230|#x2F|#47);/g,Sy={"&amp;":"&","&#38;":"&","&lt;":"<","&#60;":"<","&gt;":">","&#62;":">","&apos;":"'","&#39;":"'","&quot;":'"',"&#34;":'"',"&nbsp;":" ","&#160;":" ","&copy;":"©","&#169;":"©","&reg;":"®","&#174;":"®","&hellip;":"…","&#8230;":"…","&#x2F;":"/","&#47;":"/"},Ry=r=>Sy[r],Ay=r=>r.replace(by,Ry);let sd={bindI18n:"languageChanged",bindI18nStore:"",transEmptyNodeValue:"",transSupportBasicHtmlNodes:!0,transWrapTextNodes:"",transKeepBasicHtmlNodesFor:["br","strong","i","p"],useSuspense:!0,unescape:Ay,transDefaultProps:void 0};const Py=(r={})=>{sd={...sd,...r}},ky=()=>sd;let jm;const Dy=r=>{jm=r},Ty=()=>jm,_y={type:"3rdParty",init(r){Py(r.options.react),Dy(r)}},Ly=D.createContext();class Fy{constructor(){this.usedNamespaces={}}addUsedNamespaces(t){t.forEach(n=>{this.usedNamespaces[n]||(this.usedNamespaces[n]=!0)})}getUsedNamespaces(){return Object.keys(this.usedNamespaces)}}var Bu={exports:{}},ju={};/**
 * @license React
 * use-sync-external-store-shim.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Op;function Ny(){if(Op)return ju;Op=1;var r=Ll();function t(x,y){return x===y&&(x!==0||1/x===1/y)||x!==x&&y!==y}var n=typeof Object.is=="function"?Object.is:t,a=r.useState,i=r.useEffect,s=r.useLayoutEffect,c=r.useDebugValue;function u(x,y){var w=y(),E=a({inst:{value:w,getSnapshot:y}}),C=E[0].inst,S=E[1];return s(function(){C.value=w,C.getSnapshot=y,p(C)&&S({inst:C})},[x,w,y]),i(function(){return p(C)&&S({inst:C}),x(function(){p(C)&&S({inst:C})})},[x]),c(w),w}function p(x){var y=x.getSnapshot;x=x.value;try{var w=y();return!n(x,w)}catch{return!0}}function f(x,y){return y()}var h=typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"?f:u;return ju.useSyncExternalStore=r.useSyncExternalStore!==void 0?r.useSyncExternalStore:h,ju}var Mp;function Iy(){return Mp||(Mp=1,Bu.exports=Ny()),Bu.exports}var By=Iy();const jy=(r,t)=>bo(t)?t:Cy(t)&&bo(t.defaultValue)?t.defaultValue:Array.isArray(r)?r[r.length-1]:r,Oy={t:jy,ready:!1},My=()=>()=>{},ft=(r,t={})=>{var I,L,M;const{i18n:n}=t,{i18n:a,defaultNS:i}=D.useContext(Ly)||{},s=n||a||Ty();s&&!s.reportNamespaces&&(s.reportNamespaces=new Fy),s||Im(s,"NO_I18NEXT_INSTANCE","useTranslation: You will need to pass in an i18next instance by using initReactI18next");const c=D.useMemo(()=>{var H;return{...ky(),...(H=s==null?void 0:s.options)==null?void 0:H.react,...t}},[s,t]),{useSuspense:u,keyPrefix:p}=c,f=i||((I=s==null?void 0:s.options)==null?void 0:I.defaultNS),h=bo(f)?[f]:f||["translation"],x=D.useMemo(()=>h,h);(M=(L=s==null?void 0:s.reportNamespaces)==null?void 0:L.addUsedNamespaces)==null||M.call(L,x);const y=D.useRef(0),w=D.useCallback(H=>{if(!s)return My;const{bindI18n:z,bindI18nStore:Z}=c,oe=()=>{y.current+=1,H()};return z&&s.on(z,oe),Z&&s.store.on(Z,oe),()=>{z&&z.split(" ").forEach(re=>s.off(re,oe)),Z&&Z.split(" ").forEach(re=>s.store.off(re,oe))}},[s,c]),E=D.useRef(),C=D.useCallback(()=>{if(!s)return Oy;const H=!!(s.isInitialized||s.initializedStoreOnce)&&x.every(Y=>Ey(Y,s,c)),z=t.lng||s.language,Z=y.current,oe=E.current;if(oe&&oe.ready===H&&oe.lng===z&&oe.keyPrefix===p&&oe.revision===Z)return oe;const se={t:s.getFixedT(z,c.nsMode==="fallback"?x:x[0],p),ready:H,lng:z,keyPrefix:p,revision:Z};return E.current=se,se},[s,x,p,c,t.lng]),[S,P]=D.useState(0),{t:b,ready:A}=By.useSyncExternalStore(w,C,C);D.useEffect(()=>{if(s&&!A&&!u){const H=()=>P(z=>z+1);t.lng?jp(s,t.lng,x,H):id(s,x,H)}},[s,t.lng,x,A,u,S]);const T=s||{},B=D.useRef(null),F=D.useRef(),k=H=>{const z=Object.getOwnPropertyDescriptors(H);z.__original&&delete z.__original;const Z=Object.create(Object.getPrototypeOf(H),z);if(!Object.prototype.hasOwnProperty.call(Z,"__original"))try{Object.defineProperty(Z,"__original",{value:H,writable:!1,enumerable:!1,configurable:!1})}catch{}return Z},N=D.useMemo(()=>{const H=T,z=H==null?void 0:H.language;let Z=H;H&&(B.current&&B.current.__original===H?F.current!==z?(Z=k(H),B.current=Z,F.current=z):Z=B.current:(Z=k(H),B.current=Z,F.current=z));const oe=[b,Z,A];return oe.t=b,oe.i18n=Z,oe.ready=A,oe},[b,T,A,T.resolvedLanguage,T.language,T.languages]);if(s&&u&&!A)throw new Promise(H=>{const z=()=>H();t.lng?jp(s,t.lng,x,z):id(s,x,z)});return N},$y={"settings.title":"设置","settings.clarify.label":"需求澄清","settings.clarify.description":"启用后，系统会在生成前询问澄清问题以更好理解您的需求","settings.log.title":"系统日志","settings.log.description":"导出系统日志和 LLM 代码记录，用于调试分析","settings.log.export":"导出日志","settings.close":"关闭","settings.language.label":"界面语言","chat.motionUpdated":"已根据您的要求更新动效。","chat.motionGenerated":"已为您生成动效，可以在右侧预览区查看。您可以继续描述来调整动效，或使用参数面板进行微调。","chat.generateError":"生成动效时发生错误","chat.generateErrorDetail":"抱歉，生成动效时遇到问题：{{message}}","chat.fixSuccess":"已自动修复代码错误，动效已更新。","chat.fixError":"修复时发生错误","chat.fixErrorDetail":"修复尝试失败：{{message}}","chat.importSuccess":"已导入示例对话，可以查看参考效果","chat.importFailed":"导入失败","chat.importError":"导入示例对话失败","chat.configRequired":"请先配置 LLM API","chat.configError":"无法获取 LLM 配置，请检查配置是否有效","chat.emptyTitle":"欢迎来到 Neon Lab","chat.emptyDescription":"导入一个示例对话，快速了解平台功能","chat.importDemo":"导入示例对话","chat.emptyFallbackTitle":"输入描述来生成动效","chat.emptyFallbackHint":"例如：一个红色方块旋转360度","chat.analyzing":"正在分析需求","chat.generating":"正在生成动效","chat.placeholder1":"描述你想要的动效...","chat.placeholder2":"继续描述来调整效果...","chat.placeholder3":"输入新的描述来创建动效...","chat.send":"发送","chat.clarifyIntro":"为了更好地理解您的需求，我需要确认几个问题：","error.codeExecution":"代码执行时发生错误","error.retryCount":"已尝试修复 {{count}} 次，还可尝试 {{remaining}} 次","error.maxRetryTitle":"已尝试自动修复 {{max}} 次但未能解决问题。","error.maxRetrySuggestion":"建议您：","error.maxRetry1":"尝试用不同的方式描述您想要的效果","error.maxRetry2":"简化动效需求，分步骤实现","error.maxRetry3":"检查是否有特殊的技术要求导致生成困难","error.maxRetryHint":"您可以输入新的描述来重新生成动效。","error.fixing":"修复中...","error.autoFix":"自动修复","error.upload.fallback":"上传失败，请重试","error.image.invalidFormat":"当前仅支持PNG和JPEG格式","error.image.fileTooLarge":"图片文件过大，请压缩后重试（最大10MB）","error.image.dimensionTooLarge":"图片尺寸过大，将自动缩放","error.image.corruptedFile":"图片文件无法读取，请检查文件是否损坏","error.image.readError":"读取文件失败，请重试","error.video.invalidFormat":"当前仅支持 MP4 和 WebM 格式","error.video.fileTooLarge":"视频文件过大，请压缩后重试（最大 50MB）","error.video.durationTooLong":"视频时长超过限制（最长 60 秒），请裁剪后重试","error.video.resolutionTooLarge":"视频分辨率超过 4K，将自动缩放","error.video.corruptedFile":"视频文件无法读取，请检查文件是否损坏","error.video.loadError":"加载视频失败，请重试","error.attachment.invalidImageFormat":"当前仅支持 PNG、JPEG 和 WebP 格式","error.attachment.invalidDocumentFormat":"当前仅支持 TXT 和 MD 格式","error.attachment.fileTooLarge":"图片文件过大，请压缩后重试（最大 10MB）","error.attachment.emptyDocument":"文档内容为空","error.attachment.readError":"读取文件失败，请重试","error.attachment.maxAttachmentsReached":"最多支持 5 个附件","error.attachment.corruptedFile":"文件无法读取，请检查文件是否损坏","error.friendly.syntaxError":"代码语法有误，请检查括号、引号是否匹配","error.friendly.referenceError":"使用了未定义的变量或函数","error.friendly.typeError":"数据类型使用有误","error.friendly.rangeError":"数值超出有效范围","error.friendly.evalError":"代码执行错误","error.friendly.uriError":"URI 处理错误","error.friendly.default":"代码执行时发生错误","common.ok":"确定","common.cancel":"取消","common.confirm":"确认","common.back":"返回","common.retry":"重试","common.skip":"跳过","common.custom":"自定义...","time.yesterday":"昨天","time.sunday":"周日","time.monday":"周一","time.tuesday":"周二","time.wednesday":"周三","time.thursday":"周四","time.friday":"周五","time.saturday":"周六","history.editTitle":"编辑标题","history.duplicateConversation":"复制对话","history.exportConversation":"导出对话","history.deleteConversation":"删除对话","history.clickToSelect":"点击选择","history.processing":"正在处理中，无法切换","import.title":"导入结果","import.successCount":"成功导入 {{count}} 个对话","import.failed":"导入失败","import.skippedCount":"{{count}} 个对话因数据损坏被跳过","import.warnings":"警告：","import.errorDetails":"错误详情：","clarify.loadFailed":"加载问题失败，请重试或跳过澄清。","clarify.progress":"问题 {{current}}/{{total}}","clarify.skipGenerate":"跳过澄清，直接生成","clarify.customPlaceholder":"输入您的答案...","preview.title":"预览","preview.area":"预览区域","preview.areaHint":"在左侧输入描述生成动效","preview.performanceWarning":"渲染耗时过长，预览已暂停（单帧耗时 {{elapsed}}s）","preview.backgroundHint":"背景图仅预览使用，不参与导出","preview.clearBackground":"清除背景","preview.changeBackground":"更换背景","preview.uploadBackground":"上传背景图","preview.imageTypeError":"仅支持 PNG、JPG、WebP 格式的图片","preview.reset":"重置","preview.pause":"暂停","preview.play":"播放","codeWarning.title":"检测到非确定性代码","codeWarning.errorCount":"({{count}} 错误)","codeWarning.warningCount":"({{count}} 警告)","codeWarning.moreIssues":"还有 {{count}} 个问题...","codeWarning.hint":"这可能导致动画重播时效果不一致","codeWarning.dismiss":"关闭警告","demo.error.title":"出错了","demo.error.description":"加载Demo时发生错误，请刷新页面重试","demo.error.refresh":"刷新页面","demo.error.details":"错误详情","demo.selectHint":"选择一个Demo以查看","demo.loading":"加载Demo中...","demo.loadFailed":"加载失败","demo.loadFailedDescription":"无法加载Demo内容，请稍后重试","demo.empty.title":"暂无Demo效果","demo.empty.description":`这个目录下还没有任何Demo效果。
请查看其他目录或稍后再来。`,"demo.empty.descriptionShort":"当前还没有任何Demo效果。","demo.viewing":"正在查看: ","nav.home":"首页","nav.neonLab":"Neon Lab","nav.demos":"效果Demo","nav.heroSubtitle":"探索动效的无限可能","nav.neonLab.description":"代码驱动的渲染式动效，支持实时预览和参数精细调整。","nav.neonStudio.description":"以蓝图形式串联所有动效能力，提供统一的项目管理和协作空间。","demos.particleBurst.title":"粒子爆发","demos.particleBurst.description":"多彩粒子从中心向外爆发的动画效果","demos.particleBurst.category":"粒子","demos.waveMotion.title":"波浪运动","demos.waveMotion.description":"平滑的波浪动画，适合背景效果","demos.waveMotion.category":"波浪","demos.geometricShapes.title":"几何形状","demos.geometricShapes.description":"旋转缩放的几何图形动画效果","demos.geometricShapes.category":"几何","demos.textReveal.title":"文字揭示","demos.textReveal.description":"优雅的文字逐字显示动画","demos.textReveal.category":"文字","copy.suffix":"(副本)","copy.suffixN":"(副本 {{n}})","gallery.all":"全部","nav.settings":"设置","nav.openSettings":"打开设置","nav.exportAssetPack":"导出素材包","nav.exportVideo":"导出视频","panel.conversations":"对话","panel.parameters":"参数调整","panel.emptyParameters":"生成动效后显示可调参数","history.newConversation":"新对话","history.newConversationAction":"新建对话","history.processingCannotCreate":"正在处理中，无法新建","aspectRatio.label":"画面比例","aspectRatio.16_9":"16:9 (宽屏)","aspectRatio.4_3":"4:3 (标准)","aspectRatio.1_1":"1:1 (方形)","aspectRatio.9_16":"9:16 (竖屏)","aspectRatio.21_9":"21:9 (超宽)","aspectRatio.2_35_1":"2.35:1 (电影)","export.assetPackTitle":"导出素材包","export.videoTitle":"导出视频","config.title":"LLM 配置","config.add":"添加配置","config.edit":"编辑配置","config.delete":"删除配置","config.loading":"加载中...","config.empty":"暂无配置，点击上方按钮添加","config.switchHint":"点击配置项切换当前使用的 API","config.decryptFailed":"无法解密配置，这可能是因为设备发生了变化。请删除此配置并重新添加。","config.deleteConfirmMessage":'确定要删除配置"{{name}}"吗？此操作无法撤销。',"config.form.name":"配置名称","config.form.namePlaceholder":"例如: OpenAI GPT-4","config.form.nameHelper":"用于识别此配置的名称","config.form.nameRequired":"请输入配置名称","config.form.baseURL":"API 地址","config.form.baseURLHelper":"支持 OpenAI、DeepSeek 等兼容接口","config.form.baseURLRequired":"请输入 API 地址","config.form.baseURLInvalidProtocol":"请输入有效的 HTTP/HTTPS URL","config.form.baseURLInvalid":"请输入有效的 URL","config.form.apiKeyRequired":"请输入 API Key","config.form.apiKeyEditHelper":"留空保持原有密钥不变","config.form.model":"模型名称","config.form.modelPlaceholder":"gpt-4","config.form.modelHelper":"例如: gpt-4, deepseek-chat","config.form.modelRequired":"请输入模型名称","config.form.save":"保存修改","export.exporting":"正在导出...","export.resolution":"分辨率","export.frameRate":"帧率","export.duration":"动画时长: {{duration}} 秒","export.outputSize":"输出尺寸: {{width}} x {{height}}","export.outputFormatMp4":"输出格式: MP4 (H.264)","export.4kWarning":"4K 导出可能需要较长时间，请耐心等待","export.close":"关闭","export.startExport":"开始导出","export.failed":"导出失败","exportAssetPack.subtitle":"生成自包含的 HTML 文件（支持 MP4 导出）","exportAssetPack.filename":"文件名","exportAssetPack.customTitle":"自定义标题（可选）","exportAssetPack.customTitlePlaceholder":"留空则使用文件名","exportAssetPack.showTitle":"在导出的 HTML 中显示标题","exportAssetPack.selectParameters":"选择要导出的参数","exportAssetPack.estimatedSize":"预估文件大小:","exportAssetPack.generating":"正在生成...","exportAssetPack.downloading":"正在下载...","exportAssetPack.selectedCount":"将导出 {{count}} 个可调参数","exportAssetPack.noParameters":"将导出纯预览版本（无可调参数）","exportAssetPack.exportButton":"导出","exportAssetPack.exportingButton":"导出中...","paramSelector.empty":"当前动效没有可调参数","paramSelector.selectedCount":"已选 {{selected}}/{{total}} 个参数","paramSelector.selectAll":"全选","paramSelector.deselectAll":"取消全选","paramSelector.currentValue":"当前: {{value}}","paramSelector.type.number":"数值","paramSelector.type.color":"颜色","paramSelector.type.boolean":"开关","paramSelector.type.select":"选择","paramSelector.type.image":"图片","paramSelector.booleanOn":"开启","paramSelector.booleanOff":"关闭","paramSelector.imageValue":"(图片)"},Uy={"settings.title":"Settings","settings.clarify.label":"Clarification","settings.clarify.description":"When enabled, the system asks clarifying questions before generating","settings.log.title":"System Logs","settings.log.description":"Export system logs and LLM code records for debugging","settings.log.export":"Export Logs","settings.close":"Close","settings.language.label":"Language","chat.motionUpdated":"Motion effect updated as requested.","chat.motionGenerated":"Motion effect generated. Check the preview on the right. You can continue describing to adjust, or use the parameter panel for fine-tuning.","chat.generateError":"Error generating motion effect","chat.generateErrorDetail":"Sorry, encountered an issue: {{message}}","chat.fixSuccess":"Code error auto-fixed, motion updated.","chat.fixError":"Error during fix attempt","chat.fixErrorDetail":"Fix attempt failed: {{message}}","chat.importSuccess":"Demo conversation imported","chat.importFailed":"Import failed","chat.importError":"Failed to import demo conversation","chat.configRequired":"Please configure LLM API first","chat.configError":"Unable to get LLM config, please check if configuration is valid","chat.emptyTitle":"Welcome to Neon Lab","chat.emptyDescription":"Import a demo conversation to explore the platform","chat.importDemo":"Import Demo","chat.emptyFallbackTitle":"Describe the effect you want","chat.emptyFallbackHint":"e.g.: a red square rotating 360 degrees","chat.analyzing":"Analyzing requirements","chat.generating":"Generating motion effect","chat.placeholder1":"Describe the motion effect you want...","chat.placeholder2":"Continue describing to adjust...","chat.placeholder3":"Enter a new description...","chat.send":"Send","chat.clarifyIntro":"To better understand your needs, I have a few questions:","error.codeExecution":"Error occurred during code execution","error.retryCount":"Attempted {{count}} fix(es), {{remaining}} remaining","error.maxRetryTitle":"Auto-fix attempted {{max}} times without success.","error.maxRetrySuggestion":"Suggestions:","error.maxRetry1":"Try describing the effect in a different way","error.maxRetry2":"Simplify the requirements and build step by step","error.maxRetry3":"Check if specific technical requirements are causing issues","error.maxRetryHint":"Enter a new description to regenerate the effect.","error.fixing":"Fixing...","error.autoFix":"Auto Fix","error.upload.fallback":"Upload failed, please try again","error.image.invalidFormat":"Only PNG and JPEG formats are supported","error.image.fileTooLarge":"Image file too large, please compress (max 10MB)","error.image.dimensionTooLarge":"Image dimensions too large, will be auto-scaled","error.image.corruptedFile":"Image file cannot be read, please check if the file is corrupted","error.image.readError":"Failed to read file, please try again","error.video.invalidFormat":"Only MP4 and WebM formats are supported","error.video.fileTooLarge":"Video file too large, please compress (max 50MB)","error.video.durationTooLong":"Video duration exceeds limit (max 60 seconds), please trim","error.video.resolutionTooLarge":"Video resolution exceeds 4K, will be auto-scaled","error.video.corruptedFile":"Video file cannot be read, please check if the file is corrupted","error.video.loadError":"Failed to load video, please try again","error.attachment.invalidImageFormat":"Only PNG, JPEG, and WebP formats are supported","error.attachment.invalidDocumentFormat":"Only TXT and MD formats are supported","error.attachment.fileTooLarge":"Image file too large, please compress (max 10MB)","error.attachment.emptyDocument":"Document content is empty","error.attachment.readError":"Failed to read file, please try again","error.attachment.maxAttachmentsReached":"Maximum 5 attachments allowed","error.attachment.corruptedFile":"File cannot be read, please check if the file is corrupted","error.friendly.syntaxError":"Code syntax error, please check brackets and quotes","error.friendly.referenceError":"Undefined variable or function used","error.friendly.typeError":"Data type mismatch","error.friendly.rangeError":"Value out of valid range","error.friendly.evalError":"Code execution error","error.friendly.uriError":"URI processing error","error.friendly.default":"Error occurred during code execution","common.ok":"OK","common.cancel":"Cancel","common.confirm":"Confirm","common.back":"Back","common.retry":"Retry","common.skip":"Skip","common.custom":"Custom...","time.yesterday":"Yesterday","time.sunday":"Sun","time.monday":"Mon","time.tuesday":"Tue","time.wednesday":"Wed","time.thursday":"Thu","time.friday":"Fri","time.saturday":"Sat","history.editTitle":"Edit title","history.duplicateConversation":"Duplicate conversation","history.exportConversation":"Export conversation","history.deleteConversation":"Delete conversation","history.clickToSelect":"Click to select","history.processing":"Processing, cannot switch","import.title":"Import Results","import.successCount":"Successfully imported {{count}} conversation(s)","import.failed":"Import failed","import.skippedCount":"{{count}} conversation(s) skipped due to corrupted data","import.warnings":"Warnings:","import.errorDetails":"Error details:","clarify.loadFailed":"Failed to load questions. Please retry or skip clarification.","clarify.progress":"Question {{current}}/{{total}}","clarify.skipGenerate":"Skip clarification, generate directly","clarify.customPlaceholder":"Enter your answer...","preview.title":"Preview","preview.area":"Preview Area","preview.areaHint":"Enter a description on the left to generate effects","preview.performanceWarning":"Rendering too slow, preview paused (frame time {{elapsed}}s)","preview.backgroundHint":"Background image is for preview only, not included in export","preview.clearBackground":"Clear Background","preview.changeBackground":"Change Background","preview.uploadBackground":"Upload Background","preview.imageTypeError":"Only PNG, JPG, and WebP formats are supported","preview.reset":"Reset","preview.pause":"Pause","preview.play":"Play","codeWarning.title":"Non-deterministic code detected","codeWarning.errorCount":"({{count}} error(s))","codeWarning.warningCount":"({{count}} warning(s))","codeWarning.moreIssues":"{{count}} more issue(s)...","codeWarning.hint":"This may cause inconsistent animation on replay","codeWarning.dismiss":"Dismiss warning","demo.error.title":"Something went wrong","demo.error.description":"An error occurred while loading the demo. Please refresh the page.","demo.error.refresh":"Refresh Page","demo.error.details":"Error Details","demo.selectHint":"Select a demo to view","demo.loading":"Loading demo...","demo.loadFailed":"Load Failed","demo.loadFailedDescription":"Unable to load demo content. Please try again later.","demo.empty.title":"No Demos Available","demo.empty.description":`There are no demo effects in this directory yet.
Please check other directories or come back later.`,"demo.empty.descriptionShort":"No demo effects available yet.","demo.viewing":"Viewing: ","nav.home":"Home","nav.neonLab":"Neon Lab","nav.demos":"Demos","nav.heroSubtitle":"Endless possibilities for motion effects","nav.neonLab.description":"Code-driven rendered motion effects with real-time preview and fine parameter tuning.","nav.neonStudio.description":"Blueprint-style orchestration of all motion capabilities with unified project management.","demos.particleBurst.title":"Particle Burst","demos.particleBurst.description":"Colorful particles bursting outward from center","demos.particleBurst.category":"Particles","demos.waveMotion.title":"Wave Motion","demos.waveMotion.description":"Smooth wave animation, great for backgrounds","demos.waveMotion.category":"Waves","demos.geometricShapes.title":"Geometric Shapes","demos.geometricShapes.description":"Rotating and scaling geometric shape animations","demos.geometricShapes.category":"Geometry","demos.textReveal.title":"Text Reveal","demos.textReveal.description":"Elegant character-by-character text reveal animation","demos.textReveal.category":"Text","copy.suffix":"(Copy)","copy.suffixN":"(Copy {{n}})","gallery.all":"All","nav.settings":"Settings","nav.openSettings":"Open Settings","nav.exportAssetPack":"Export Assets","nav.exportVideo":"Export Video","panel.conversations":"Conversations","panel.parameters":"Parameters","panel.emptyParameters":"Adjustable parameters shown after generating effects","history.newConversation":"New Conversation","history.newConversationAction":"New Conversation","history.processingCannotCreate":"Processing, cannot create new","aspectRatio.label":"Aspect Ratio","aspectRatio.16_9":"16:9 (Widescreen)","aspectRatio.4_3":"4:3 (Standard)","aspectRatio.1_1":"1:1 (Square)","aspectRatio.9_16":"9:16 (Portrait)","aspectRatio.21_9":"21:9 (Ultra-wide)","aspectRatio.2_35_1":"2.35:1 (Cinema)","export.assetPackTitle":"Export Assets","export.videoTitle":"Export Video","config.title":"LLM Configurations","config.add":"Add Config","config.edit":"Edit Config","config.delete":"Delete Config","config.loading":"Loading...","config.empty":"No configurations yet, click the button above to add one","config.switchHint":"Click a config to switch the active API","config.decryptFailed":"Unable to decrypt config. This may be due to a device change. Please delete and re-add this config.","config.deleteConfirmMessage":'Are you sure you want to delete "{{name}}"? This action cannot be undone.',"config.form.name":"Config Name","config.form.namePlaceholder":"e.g. OpenAI GPT-4","config.form.nameHelper":"A name to identify this configuration","config.form.nameRequired":"Please enter a config name","config.form.baseURL":"API URL","config.form.baseURLHelper":"Supports OpenAI, DeepSeek and compatible APIs","config.form.baseURLRequired":"Please enter an API URL","config.form.baseURLInvalidProtocol":"Please enter a valid HTTP/HTTPS URL","config.form.baseURLInvalid":"Please enter a valid URL","config.form.apiKeyRequired":"Please enter an API Key","config.form.apiKeyEditHelper":"Leave empty to keep the existing key","config.form.model":"Model Name","config.form.modelPlaceholder":"gpt-4","config.form.modelHelper":"e.g. gpt-4, deepseek-chat","config.form.modelRequired":"Please enter a model name","config.form.save":"Save Changes","export.exporting":"Exporting...","export.resolution":"Resolution","export.frameRate":"Frame Rate","export.duration":"Duration: {{duration}}s","export.outputSize":"Output Size: {{width}} x {{height}}","export.outputFormatMp4":"Output Format: MP4 (H.264)","export.4kWarning":"4K export may take longer, please be patient","export.close":"Close","export.startExport":"Start Export","export.failed":"Export failed","exportAssetPack.subtitle":"Generate self-contained HTML file (supports MP4 export)","exportAssetPack.filename":"Filename","exportAssetPack.customTitle":"Custom Title (optional)","exportAssetPack.customTitlePlaceholder":"Leave empty to use filename","exportAssetPack.showTitle":"Show title in exported HTML","exportAssetPack.selectParameters":"Select parameters to export","exportAssetPack.estimatedSize":"Estimated file size:","exportAssetPack.generating":"Generating...","exportAssetPack.downloading":"Downloading...","exportAssetPack.selectedCount":"Will export {{count}} adjustable parameter(s)","exportAssetPack.noParameters":"Will export preview-only version (no adjustable parameters)","exportAssetPack.exportButton":"Export","exportAssetPack.exportingButton":"Exporting...","paramSelector.empty":"No adjustable parameters for this effect","paramSelector.selectedCount":"Selected {{selected}}/{{total}} parameters","paramSelector.selectAll":"Select All","paramSelector.deselectAll":"Deselect All","paramSelector.currentValue":"Current: {{value}}","paramSelector.type.number":"Number","paramSelector.type.color":"Color","paramSelector.type.boolean":"Toggle","paramSelector.type.select":"Select","paramSelector.type.image":"Image","paramSelector.booleanOn":"On","paramSelector.booleanOff":"Off","paramSelector.imageValue":"(Image)"};Ct.use(_y).init({resources:{zh:{translation:$y},en:{translation:Uy}},lng:localStorage.getItem("neon-locale")||(navigator.language.startsWith("en")?"en":"zh"),fallbackLng:"zh",interpolation:{escapeValue:!1}});/**
 * react-router v7.12.0
 *
 * Copyright (c) Remix Software Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE.md file in the root directory of this source tree.
 *
 * @license MIT
 */var Om=r=>{throw TypeError(r)},zy=(r,t,n)=>t.has(r)||Om("Cannot "+n),Ou=(r,t,n)=>(zy(r,t,"read from private field"),n?n.call(r):t.get(r)),Hy=(r,t,n)=>t.has(r)?Om("Cannot add the same private member more than once"):t instanceof WeakSet?t.add(r):t.set(r,n),$p="popstate";function Vy(r={}){function t(i,s){let{pathname:c="/",search:u="",hash:p=""}=fn(i.location.hash.substring(1));return!c.startsWith("/")&&!c.startsWith(".")&&(c="/"+c),ti("",{pathname:c,search:u,hash:p},s.state&&s.state.usr||null,s.state&&s.state.key||"default")}function n(i,s){let c=i.document.querySelector("base"),u="";if(c&&c.getAttribute("href")){let p=i.location.href,f=p.indexOf("#");u=f===-1?p:p.slice(0,f)}return u+"#"+(typeof s=="string"?s:qr(s))}function a(i,s){bt(i.pathname.charAt(0)==="/",`relative pathnames are not supported in hash history.push(${JSON.stringify(s)})`)}return Gy(t,n,a,r)}function qe(r,t){if(r===!1||r===null||typeof r>"u")throw new Error(t)}function bt(r,t){if(!r){typeof console<"u"&&console.warn(t);try{throw new Error(t)}catch{}}}function Wy(){return Math.random().toString(36).substring(2,10)}function Up(r,t){return{usr:r.state,key:r.key,idx:t}}function ti(r,t,n=null,a){return{pathname:typeof r=="string"?r:r.pathname,search:"",hash:"",...typeof t=="string"?fn(t):t,state:n,key:t&&t.key||a||Wy()}}function qr({pathname:r="/",search:t="",hash:n=""}){return t&&t!=="?"&&(r+=t.charAt(0)==="?"?t:"?"+t),n&&n!=="#"&&(r+=n.charAt(0)==="#"?n:"#"+n),r}function fn(r){let t={};if(r){let n=r.indexOf("#");n>=0&&(t.hash=r.substring(n),r=r.substring(0,n));let a=r.indexOf("?");a>=0&&(t.search=r.substring(a),r=r.substring(0,a)),r&&(t.pathname=r)}return t}function Gy(r,t,n,a={}){let{window:i=document.defaultView,v5Compat:s=!1}=a,c=i.history,u="POP",p=null,f=h();f==null&&(f=0,c.replaceState({...c.state,idx:f},""));function h(){return(c.state||{idx:null}).idx}function x(){u="POP";let S=h(),P=S==null?null:S-f;f=S,p&&p({action:u,location:C.location,delta:P})}function y(S,P){u="PUSH";let b=ti(C.location,S,P);n&&n(b,S),f=h()+1;let A=Up(b,f),T=C.createHref(b);try{c.pushState(A,"",T)}catch(B){if(B instanceof DOMException&&B.name==="DataCloneError")throw B;i.location.assign(T)}s&&p&&p({action:u,location:C.location,delta:1})}function w(S,P){u="REPLACE";let b=ti(C.location,S,P);n&&n(b,S),f=h();let A=Up(b,f),T=C.createHref(b);c.replaceState(A,"",T),s&&p&&p({action:u,location:C.location,delta:0})}function E(S){return Mm(S)}let C={get action(){return u},get location(){return r(i,c)},listen(S){if(p)throw new Error("A history only accepts one active listener");return i.addEventListener($p,x),p=S,()=>{i.removeEventListener($p,x),p=null}},createHref(S){return t(i,S)},createURL:E,encodeLocation(S){let P=E(S);return{pathname:P.pathname,search:P.search,hash:P.hash}},push:y,replace:w,go(S){return c.go(S)}};return C}function Mm(r,t=!1){let n="http://localhost";typeof window<"u"&&(n=window.location.origin!=="null"?window.location.origin:window.location.href),qe(n,"No window.location.(origin|href) available to create URL");let a=typeof r=="string"?r:qr(r);return a=a.replace(/ $/,"%20"),!t&&a.startsWith("//")&&(a=n+a),new URL(a,n)}var Ga,zp=class{constructor(r){if(Hy(this,Ga,new Map),r)for(let[t,n]of r)this.set(t,n)}get(r){if(Ou(this,Ga).has(r))return Ou(this,Ga).get(r);if(r.defaultValue!==void 0)return r.defaultValue;throw new Error("No value found for context")}set(r,t){Ou(this,Ga).set(r,t)}};Ga=new WeakMap;var qy=new Set(["lazy","caseSensitive","path","id","index","children"]);function Ky(r){return qy.has(r)}var Xy=new Set(["lazy","caseSensitive","path","id","index","middleware","children"]);function Yy(r){return Xy.has(r)}function Qy(r){return r.index===!0}function ri(r,t,n=[],a={},i=!1){return r.map((s,c)=>{let u=[...n,String(c)],p=typeof s.id=="string"?s.id:u.join("-");if(qe(s.index!==!0||!s.children,"Cannot specify children on an index route"),qe(i||!a[p],`Found a route id collision on id "${p}".  Route id's must be globally unique within Data Router usages`),Qy(s)){let f={...s,id:p};return a[p]=Hp(f,t(f)),f}else{let f={...s,id:p,children:void 0};return a[p]=Hp(f,t(f)),s.children&&(f.children=ri(s.children,t,u,a,i)),f}})}function Hp(r,t){return Object.assign(r,{...t,...typeof t.lazy=="object"&&t.lazy!=null?{lazy:{...r.lazy,...t.lazy}}:{}})}function Gn(r,t,n="/"){return qa(r,t,n,!1)}function qa(r,t,n,a){let i=typeof t=="string"?fn(t):t,s=Rr(i.pathname||"/",n);if(s==null)return null;let c=$m(r);Zy(c);let u=null;for(let p=0;u==null&&p<c.length;++p){let f=u1(s);u=l1(c[p],f,a)}return u}function Jy(r,t){let{route:n,pathname:a,params:i}=r;return{id:n.id,pathname:a,params:i,data:t[n.id],loaderData:t[n.id],handle:n.handle}}function $m(r,t=[],n=[],a="",i=!1){let s=(c,u,p=i,f)=>{let h={relativePath:f===void 0?c.path||"":f,caseSensitive:c.caseSensitive===!0,childrenIndex:u,route:c};if(h.relativePath.startsWith("/")){if(!h.relativePath.startsWith(a)&&p)return;qe(h.relativePath.startsWith(a),`Absolute route path "${h.relativePath}" nested under path "${a}" is not valid. An absolute child route path must start with the combined path of all its parent routes.`),h.relativePath=h.relativePath.slice(a.length)}let x=Gr([a,h.relativePath]),y=n.concat(h);c.children&&c.children.length>0&&(qe(c.index!==!0,`Index routes must not have child routes. Please remove all child routes from route path "${x}".`),$m(c.children,t,y,x,p)),!(c.path==null&&!c.index)&&t.push({path:x,score:i1(x,c.index),routesMeta:y})};return r.forEach((c,u)=>{var p;if(c.path===""||!((p=c.path)!=null&&p.includes("?")))s(c,u);else for(let f of Um(c.path))s(c,u,!0,f)}),t}function Um(r){let t=r.split("/");if(t.length===0)return[];let[n,...a]=t,i=n.endsWith("?"),s=n.replace(/\?$/,"");if(a.length===0)return i?[s,""]:[s];let c=Um(a.join("/")),u=[];return u.push(...c.map(p=>p===""?s:[s,p].join("/"))),i&&u.push(...c),u.map(p=>r.startsWith("/")&&p===""?"/":p)}function Zy(r){r.sort((t,n)=>t.score!==n.score?n.score-t.score:s1(t.routesMeta.map(a=>a.childrenIndex),n.routesMeta.map(a=>a.childrenIndex)))}var e1=/^:[\w-]+$/,t1=3,r1=2,n1=1,o1=10,a1=-2,Vp=r=>r==="*";function i1(r,t){let n=r.split("/"),a=n.length;return n.some(Vp)&&(a+=a1),t&&(a+=r1),n.filter(i=>!Vp(i)).reduce((i,s)=>i+(e1.test(s)?t1:s===""?n1:o1),a)}function s1(r,t){return r.length===t.length&&r.slice(0,-1).every((a,i)=>a===t[i])?r[r.length-1]-t[t.length-1]:0}function l1(r,t,n=!1){let{routesMeta:a}=r,i={},s="/",c=[];for(let u=0;u<a.length;++u){let p=a[u],f=u===a.length-1,h=s==="/"?t:t.slice(s.length)||"/",x=Sl({path:p.relativePath,caseSensitive:p.caseSensitive,end:f},h),y=p.route;if(!x&&f&&n&&!a[a.length-1].route.index&&(x=Sl({path:p.relativePath,caseSensitive:p.caseSensitive,end:!1},h)),!x)return null;Object.assign(i,x.params),c.push({params:i,pathname:Gr([s,x.pathname]),pathnameBase:p1(Gr([s,x.pathnameBase])),route:y}),x.pathnameBase!=="/"&&(s=Gr([s,x.pathnameBase]))}return c}function Sl(r,t){typeof r=="string"&&(r={path:r,caseSensitive:!1,end:!0});let[n,a]=c1(r.path,r.caseSensitive,r.end),i=t.match(n);if(!i)return null;let s=i[0],c=s.replace(/(.)\/+$/,"$1"),u=i.slice(1);return{params:a.reduce((f,{paramName:h,isOptional:x},y)=>{if(h==="*"){let E=u[y]||"";c=s.slice(0,s.length-E.length).replace(/(.)\/+$/,"$1")}const w=u[y];return x&&!w?f[h]=void 0:f[h]=(w||"").replace(/%2F/g,"/"),f},{}),pathname:s,pathnameBase:c,pattern:r}}function c1(r,t=!1,n=!0){bt(r==="*"||!r.endsWith("*")||r.endsWith("/*"),`Route path "${r}" will be treated as if it were "${r.replace(/\*$/,"/*")}" because the \`*\` character must always follow a \`/\` in the pattern. To get rid of this warning, please change the route path to "${r.replace(/\*$/,"/*")}".`);let a=[],i="^"+r.replace(/\/*\*?$/,"").replace(/^\/*/,"/").replace(/[\\.*+^${}|()[\]]/g,"\\$&").replace(/\/:([\w-]+)(\?)?/g,(c,u,p)=>(a.push({paramName:u,isOptional:p!=null}),p?"/?([^\\/]+)?":"/([^\\/]+)")).replace(/\/([\w-]+)\?(\/|$)/g,"(/$1)?$2");return r.endsWith("*")?(a.push({paramName:"*"}),i+=r==="*"||r==="/*"?"(.*)$":"(?:\\/(.+)|\\/*)$"):n?i+="\\/*$":r!==""&&r!=="/"&&(i+="(?:(?=\\/|$))"),[new RegExp(i,t?void 0:"i"),a]}function u1(r){try{return r.split("/").map(t=>decodeURIComponent(t).replace(/\//g,"%2F")).join("/")}catch(t){return bt(!1,`The URL path "${r}" could not be decoded because it is a malformed URL segment. This is probably due to a bad percent encoding (${t}).`),r}}function Rr(r,t){if(t==="/")return r;if(!r.toLowerCase().startsWith(t.toLowerCase()))return null;let n=t.endsWith("/")?t.length-1:t.length,a=r.charAt(n);return a&&a!=="/"?null:r.slice(n)||"/"}function d1({basename:r,pathname:t}){return t==="/"?r:Gr([r,t])}var zm=/^(?:[a-z][a-z0-9+.-]*:|\/\/)/i,Nl=r=>zm.test(r);function f1(r,t="/"){let{pathname:n,search:a="",hash:i=""}=typeof r=="string"?fn(r):r,s;if(n)if(Nl(n))s=n;else{if(n.includes("//")){let c=n;n=n.replace(/\/\/+/g,"/"),bt(!1,`Pathnames cannot have embedded double slashes - normalizing ${c} -> ${n}`)}n.startsWith("/")?s=Wp(n.substring(1),"/"):s=Wp(n,t)}else s=t;return{pathname:s,search:h1(a),hash:m1(i)}}function Wp(r,t){let n=t.replace(/\/+$/,"").split("/");return r.split("/").forEach(i=>{i===".."?n.length>1&&n.pop():i!=="."&&n.push(i)}),n.length>1?n.join("/"):"/"}function Mu(r,t,n,a){return`Cannot include a '${r}' character in a manually specified \`to.${t}\` field [${JSON.stringify(a)}].  Please separate it out to the \`to.${n}\` field. Alternatively you may provide the full path as a string in <Link to="..."> and the router will parse it for you.`}function Hm(r){return r.filter((t,n)=>n===0||t.route.path&&t.route.path.length>0)}function yd(r){let t=Hm(r);return t.map((n,a)=>a===t.length-1?n.pathname:n.pathnameBase)}function wd(r,t,n,a=!1){let i;typeof r=="string"?i=fn(r):(i={...r},qe(!i.pathname||!i.pathname.includes("?"),Mu("?","pathname","search",i)),qe(!i.pathname||!i.pathname.includes("#"),Mu("#","pathname","hash",i)),qe(!i.search||!i.search.includes("#"),Mu("#","search","hash",i)));let s=r===""||i.pathname==="",c=s?"/":i.pathname,u;if(c==null)u=n;else{let x=t.length-1;if(!a&&c.startsWith("..")){let y=c.split("/");for(;y[0]==="..";)y.shift(),x-=1;i.pathname=y.join("/")}u=x>=0?t[x]:"/"}let p=f1(i,u),f=c&&c!=="/"&&c.endsWith("/"),h=(s||c===".")&&n.endsWith("/");return!p.pathname.endsWith("/")&&(f||h)&&(p.pathname+="/"),p}var Gr=r=>r.join("/").replace(/\/\/+/g,"/"),p1=r=>r.replace(/\/+$/,"").replace(/^\/*/,"/"),h1=r=>!r||r==="?"?"":r.startsWith("?")?r:"?"+r,m1=r=>!r||r==="#"?"":r.startsWith("#")?r:"#"+r,li=class{constructor(r,t,n,a=!1){this.status=r,this.statusText=t||"",this.internal=a,n instanceof Error?(this.data=n.toString(),this.error=n):this.data=n}};function ni(r){return r!=null&&typeof r.status=="number"&&typeof r.statusText=="string"&&typeof r.internal=="boolean"&&"data"in r}function ci(r){return r.map(t=>t.route.path).filter(Boolean).join("/").replace(/\/\/*/g,"/")||"/"}var Vm=typeof window<"u"&&typeof window.document<"u"&&typeof window.document.createElement<"u";function Wm(r,t){let n=r;if(typeof n!="string"||!zm.test(n))return{absoluteURL:void 0,isExternal:!1,to:n};let a=n,i=!1;if(Vm)try{let s=new URL(window.location.href),c=n.startsWith("//")?new URL(s.protocol+n):new URL(n),u=Rr(c.pathname,t);c.origin===s.origin&&u!=null?n=u+c.search+c.hash:i=!0}catch{bt(!1,`<Link to="${n}"> contains an invalid URL which will probably break when clicked - please update to a valid URL path.`)}return{absoluteURL:a,isExternal:i,to:n}}var Kn=Symbol("Uninstrumented");function g1(r,t){let n={lazy:[],"lazy.loader":[],"lazy.action":[],"lazy.middleware":[],middleware:[],loader:[],action:[]};r.forEach(i=>i({id:t.id,index:t.index,path:t.path,instrument(s){let c=Object.keys(n);for(let u of c)s[u]&&n[u].push(s[u])}}));let a={};if(typeof t.lazy=="function"&&n.lazy.length>0){let i=ta(n.lazy,t.lazy,()=>{});i&&(a.lazy=i)}if(typeof t.lazy=="object"){let i=t.lazy;["middleware","loader","action"].forEach(s=>{let c=i[s],u=n[`lazy.${s}`];if(typeof c=="function"&&u.length>0){let p=ta(u,c,()=>{});p&&(a.lazy=Object.assign(a.lazy||{},{[s]:p}))}})}return["loader","action"].forEach(i=>{let s=t[i];if(typeof s=="function"&&n[i].length>0){let c=s[Kn]??s,u=ta(n[i],c,(...p)=>Gp(p[0]));u&&(i==="loader"&&c.hydrate===!0&&(u.hydrate=!0),u[Kn]=c,a[i]=u)}}),t.middleware&&t.middleware.length>0&&n.middleware.length>0&&(a.middleware=t.middleware.map(i=>{let s=i[Kn]??i,c=ta(n.middleware,s,(...u)=>Gp(u[0]));return c?(c[Kn]=s,c):i})),a}function x1(r,t){let n={navigate:[],fetch:[]};if(t.forEach(a=>a({instrument(i){let s=Object.keys(i);for(let c of s)i[c]&&n[c].push(i[c])}})),n.navigate.length>0){let a=r.navigate[Kn]??r.navigate,i=ta(n.navigate,a,(...s)=>{let[c,u]=s;return{to:typeof c=="number"||typeof c=="string"?c:c?qr(c):".",...qp(r,u??{})}});i&&(i[Kn]=a,r.navigate=i)}if(n.fetch.length>0){let a=r.fetch[Kn]??r.fetch,i=ta(n.fetch,a,(...s)=>{let[c,,u,p]=s;return{href:u??".",fetcherKey:c,...qp(r,p??{})}});i&&(i[Kn]=a,r.fetch=i)}return r}function ta(r,t,n){return r.length===0?null:async(...a)=>{let i=await Gm(r,n(...a),()=>t(...a),r.length-1);if(i.type==="error")throw i.value;return i.value}}async function Gm(r,t,n,a){let i=r[a],s;if(i){let c,u=async()=>(c?console.error("You cannot call instrumented handlers more than once"):c=Gm(r,t,n,a-1),s=await c,qe(s,"Expected a result"),s.type==="error"&&s.value instanceof Error?{status:"error",error:s.value}:{status:"success",error:void 0});try{await i(u,t)}catch(p){console.error("An instrumentation function threw an error:",p)}c||await u(),await c}else try{s={type:"success",value:await n()}}catch(c){s={type:"error",value:c}}return s||{type:"error",value:new Error("No result assigned in instrumentation chain.")}}function Gp(r){let{request:t,context:n,params:a,unstable_pattern:i}=r;return{request:v1(t),params:{...a},unstable_pattern:i,context:y1(n)}}function qp(r,t){return{currentUrl:qr(r.state.location),..."formMethod"in t?{formMethod:t.formMethod}:{},..."formEncType"in t?{formEncType:t.formEncType}:{},..."formData"in t?{formData:t.formData}:{},..."body"in t?{body:t.body}:{}}}function v1(r){return{method:r.method,url:r.url,headers:{get:(...t)=>r.headers.get(...t)}}}function y1(r){if(E1(r)){let t={...r};return Object.freeze(t),t}else return{get:t=>r.get(t)}}var w1=Object.getOwnPropertyNames(Object.prototype).sort().join("\0");function E1(r){if(r===null||typeof r!="object")return!1;const t=Object.getPrototypeOf(r);return t===Object.prototype||t===null||Object.getOwnPropertyNames(t).sort().join("\0")===w1}var qm=["POST","PUT","PATCH","DELETE"],C1=new Set(qm),b1=["GET",...qm],S1=new Set(b1),Km=new Set([301,302,303,307,308]),R1=new Set([307,308]),$u={state:"idle",location:void 0,formMethod:void 0,formAction:void 0,formEncType:void 0,formData:void 0,json:void 0,text:void 0},A1={state:"idle",data:void 0,formMethod:void 0,formAction:void 0,formEncType:void 0,formData:void 0,json:void 0,text:void 0},Ua={state:"unblocked",proceed:void 0,reset:void 0,location:void 0},P1=r=>({hasErrorBoundary:!!r.hasErrorBoundary}),Xm="remix-router-transitions",Ym=Symbol("ResetLoaderData");function k1(r){const t=r.window?r.window:typeof window<"u"?window:void 0,n=typeof t<"u"&&typeof t.document<"u"&&typeof t.document.createElement<"u";qe(r.routes.length>0,"You must provide a non-empty routes array to createRouter");let a=r.hydrationRouteProperties||[],i=r.mapRouteProperties||P1,s=i;if(r.unstable_instrumentations){let O=r.unstable_instrumentations;s=W=>({...i(W),...g1(O.map(Q=>Q.route).filter(Boolean),W)})}let c={},u=ri(r.routes,s,void 0,c),p,f=r.basename||"/";f.startsWith("/")||(f=`/${f}`);let h=r.dataStrategy||F1,x={...r.future},y=null,w=new Set,E=null,C=null,S=null,P=r.hydrationData!=null,b=Gn(u,r.history.location,f),A=!1,T=null,B;if(b==null&&!r.patchRoutesOnNavigation){let O=Sr(404,{pathname:r.history.location.pathname}),{matches:W,route:Q}=xs(u);B=!0,b=W,T={[Q.id]:O}}else if(b&&!r.hydrationData&&Kr(b,u,r.history.location.pathname).active&&(b=null),b)if(b.some(O=>O.route.lazy))B=!1;else if(!b.some(O=>Ed(O.route)))B=!0;else{let O=r.hydrationData?r.hydrationData.loaderData:null,W=r.hydrationData?r.hydrationData.errors:null;if(W){let Q=b.findIndex(ae=>W[ae.route.id]!==void 0);B=b.slice(0,Q+1).every(ae=>!cd(ae.route,O,W))}else B=b.every(Q=>!cd(Q.route,O,W))}else{B=!1,b=[];let O=Kr(null,u,r.history.location.pathname);O.active&&O.matches&&(A=!0,b=O.matches)}let F,k={historyAction:r.history.action,location:r.history.location,matches:b,initialized:B,navigation:$u,restoreScrollPosition:r.hydrationData!=null?!1:null,preventScrollReset:!1,revalidation:"idle",loaderData:r.hydrationData&&r.hydrationData.loaderData||{},actionData:r.hydrationData&&r.hydrationData.actionData||null,errors:r.hydrationData&&r.hydrationData.errors||T,fetchers:new Map,blockers:new Map},N="POP",I=null,L=!1,M,H=!1,z=new Map,Z=null,oe=!1,re=!1,se=new Set,Y=new Map,te=0,ee=-1,_=new Map,$=new Set,G=new Map,K=new Map,ue=new Set,me=new Map,Le,he=null;function Ve(){if(y=r.history.listen(({action:O,location:W,delta:Q})=>{if(Le){Le(),Le=void 0;return}bt(me.size===0||Q!=null,"You are trying to use a blocker on a POP navigation to a location that was not created by @remix-run/router. This will fail silently in production. This can happen if you are navigating outside the router via `window.history.pushState`/`window.location.hash` instead of using router navigation APIs.  This can also happen if you are using createHashRouter and the user manually changes the URL.");let ae=kr({currentLocation:k.location,nextLocation:W,historyAction:O});if(ae&&Q!=null){let fe=new Promise(De=>{Le=De});r.history.go(Q*-1),Pr(ae,{state:"blocked",location:W,proceed(){Pr(ae,{state:"proceeding",proceed:void 0,reset:void 0,location:W}),fe.then(()=>r.history.go(Q))},reset(){let De=new Map(k.blockers);De.set(ae,Ua),We({blockers:De})}}),I==null||I.resolve(),I=null;return}return Ye(O,W)}),n){Q1(t,z);let O=()=>J1(t,z);t.addEventListener("pagehide",O),Z=()=>t.removeEventListener("pagehide",O)}return k.initialized||Ye("POP",k.location,{initialHydration:!0}),F}function ot(){y&&y(),Z&&Z(),w.clear(),M&&M.abort(),k.fetchers.forEach((O,W)=>cr(W)),k.blockers.forEach((O,W)=>vn(W))}function Vt(O){return w.add(O),()=>w.delete(O)}function We(O,W={}){O.matches&&(O.matches=O.matches.map(fe=>{let De=c[fe.route.id],Ne=fe.route;return Ne.element!==De.element||Ne.errorElement!==De.errorElement||Ne.hydrateFallbackElement!==De.hydrateFallbackElement?{...fe,route:De}:fe})),k={...k,...O};let Q=[],ae=[];k.fetchers.forEach((fe,De)=>{fe.state==="idle"&&(ue.has(De)?Q.push(De):ae.push(De))}),ue.forEach(fe=>{!k.fetchers.has(fe)&&!Y.has(fe)&&Q.push(fe)}),[...w].forEach(fe=>fe(k,{deletedFetchers:Q,newErrors:O.errors??null,viewTransitionOpts:W.viewTransitionOpts,flushSync:W.flushSync===!0})),Q.forEach(fe=>cr(fe)),ae.forEach(fe=>k.fetchers.delete(fe))}function Ee(O,W,{flushSync:Q}={}){var Ue,_e;let ae=k.actionData!=null&&k.navigation.formMethod!=null&&Xt(k.navigation.formMethod)&&k.navigation.state==="loading"&&((Ue=O.state)==null?void 0:Ue._isRedirect)!==!0,fe;W.actionData?Object.keys(W.actionData).length>0?fe=W.actionData:fe=null:ae?fe=k.actionData:fe=null;let De=W.loaderData?nh(k.loaderData,W.loaderData,W.matches||[],W.errors):k.loaderData,Ne=k.blockers;Ne.size>0&&(Ne=new Map(Ne),Ne.forEach((ze,Xe)=>Ne.set(Xe,Ua)));let Ce=oe?!1:yn(O,W.matches||k.matches),Se=L===!0||k.navigation.formMethod!=null&&Xt(k.navigation.formMethod)&&((_e=O.state)==null?void 0:_e._isRedirect)!==!0;p&&(u=p,p=void 0),oe||N==="POP"||(N==="PUSH"?r.history.push(O,O.state):N==="REPLACE"&&r.history.replace(O,O.state));let Ae;if(N==="POP"){let ze=z.get(k.location.pathname);ze&&ze.has(O.pathname)?Ae={currentLocation:k.location,nextLocation:O}:z.has(O.pathname)&&(Ae={currentLocation:O,nextLocation:k.location})}else if(H){let ze=z.get(k.location.pathname);ze?ze.add(O.pathname):(ze=new Set([O.pathname]),z.set(k.location.pathname,ze)),Ae={currentLocation:k.location,nextLocation:O}}We({...W,actionData:fe,loaderData:De,historyAction:N,location:O,initialized:!0,navigation:$u,revalidation:"idle",restoreScrollPosition:Ce,preventScrollReset:Se,blockers:Ne},{viewTransitionOpts:Ae,flushSync:Q===!0}),N="POP",L=!1,H=!1,oe=!1,re=!1,I==null||I.resolve(),I=null,he==null||he.resolve(),he=null}async function ge(O,W){if(I==null||I.resolve(),I=null,typeof O=="number"){I||(I=sh());let Xe=I.promise;return r.history.go(O),Xe}let Q=ld(k.location,k.matches,f,O,W==null?void 0:W.fromRouteId,W==null?void 0:W.relative),{path:ae,submission:fe,error:De}=Kp(!1,Q,W),Ne=k.location,Ce=ti(k.location,ae,W&&W.state);Ce={...Ce,...r.history.encodeLocation(Ce)};let Se=W&&W.replace!=null?W.replace:void 0,Ae="PUSH";Se===!0?Ae="REPLACE":Se===!1||fe!=null&&Xt(fe.formMethod)&&fe.formAction===k.location.pathname+k.location.search&&(Ae="REPLACE");let Ue=W&&"preventScrollReset"in W?W.preventScrollReset===!0:void 0,_e=(W&&W.flushSync)===!0,ze=kr({currentLocation:Ne,nextLocation:Ce,historyAction:Ae});if(ze){Pr(ze,{state:"blocked",location:Ce,proceed(){Pr(ze,{state:"proceeding",proceed:void 0,reset:void 0,location:Ce}),ge(O,W)},reset(){let Xe=new Map(k.blockers);Xe.set(ze,Ua),We({blockers:Xe})}});return}await Ye(Ae,Ce,{submission:fe,pendingError:De,preventScrollReset:Ue,replace:W&&W.replace,enableViewTransition:W&&W.viewTransition,flushSync:_e,callSiteDefaultShouldRevalidate:W&&W.unstable_defaultShouldRevalidate})}function je(){he||(he=sh()),Rt(),We({revalidation:"loading"});let O=he.promise;return k.navigation.state==="submitting"?O:k.navigation.state==="idle"?(Ye(k.historyAction,k.location,{startUninterruptedRevalidation:!0}),O):(Ye(N||k.historyAction,k.navigation.location,{overrideNavigation:k.navigation,enableViewTransition:H===!0}),O)}async function Ye(O,W,Q){M&&M.abort(),M=null,N=O,oe=(Q&&Q.startUninterruptedRevalidation)===!0,Do(k.location,k.matches),L=(Q&&Q.preventScrollReset)===!0,H=(Q&&Q.enableViewTransition)===!0;let ae=p||u,fe=Q&&Q.overrideNavigation,De=Q!=null&&Q.initialHydration&&k.matches&&k.matches.length>0&&!A?k.matches:Gn(ae,W,f),Ne=(Q&&Q.flushSync)===!0;if(De&&k.initialized&&!re&&U1(k.location,W)&&!(Q&&Q.submission&&Xt(Q.submission.formMethod))){Ee(W,{matches:De},{flushSync:Ne});return}let Ce=Kr(De,ae,W.pathname);if(Ce.active&&Ce.matches&&(De=Ce.matches),!De){let{error:At,notFoundMatches:Bt,route:nt}=ur(W.pathname);Ee(W,{matches:Bt,loaderData:{},errors:{[nt.id]:At}},{flushSync:Ne});return}M=new AbortController;let Se=ea(r.history,W,M.signal,Q&&Q.submission),Ae=r.getContext?await r.getContext():new zp,Ue;if(Q&&Q.pendingError)Ue=[qn(De).route.id,{type:"error",error:Q.pendingError}];else if(Q&&Q.submission&&Xt(Q.submission.formMethod)){let At=await tt(Se,W,Q.submission,De,Ae,Ce.active,Q&&Q.initialHydration===!0,{replace:Q.replace,flushSync:Ne});if(At.shortCircuited)return;if(At.pendingActionResult){let[Bt,nt]=At.pendingActionResult;if(gr(nt)&&ni(nt.error)&&nt.error.status===404){M=null,Ee(W,{matches:At.matches,loaderData:{},errors:{[Bt]:nt.error}});return}}De=At.matches||De,Ue=At.pendingActionResult,fe=Uu(W,Q.submission),Ne=!1,Ce.active=!1,Se=ea(r.history,Se.url,Se.signal)}let{shortCircuited:_e,matches:ze,loaderData:Xe,errors:_t}=await Dt(Se,W,De,Ae,Ce.active,fe,Q&&Q.submission,Q&&Q.fetcherSubmission,Q&&Q.replace,Q&&Q.initialHydration===!0,Ne,Ue,Q&&Q.callSiteDefaultShouldRevalidate);_e||(M=null,Ee(W,{matches:ze||De,...oh(Ue),loaderData:Xe,errors:_t}))}async function tt(O,W,Q,ae,fe,De,Ne,Ce={}){Rt();let Se=X1(W,Q);if(We({navigation:Se},{flushSync:Ce.flushSync===!0}),De){let _e=await Or(ae,W.pathname,O.signal);if(_e.type==="aborted")return{shortCircuited:!0};if(_e.type==="error"){if(_e.partialMatches.length===0){let{matches:Xe,route:_t}=xs(u);return{matches:Xe,pendingActionResult:[_t.id,{type:"error",error:_e.error}]}}let ze=qn(_e.partialMatches).route.id;return{matches:_e.partialMatches,pendingActionResult:[ze,{type:"error",error:_e.error}]}}else if(_e.matches)ae=_e.matches;else{let{notFoundMatches:ze,error:Xe,route:_t}=ur(W.pathname);return{matches:ze,pendingActionResult:[_t.id,{type:"error",error:Xe}]}}}let Ae,Ue=Ts(ae,W);if(!Ue.route.action&&!Ue.route.lazy)Ae={type:"error",error:Sr(405,{method:O.method,pathname:W.pathname,routeId:Ue.route.id})};else{let _e=ra(s,c,O,ae,Ue,Ne?[]:a,fe),ze=await ct(O,_e,fe,null);if(Ae=ze[Ue.route.id],!Ae){for(let Xe of ae)if(ze[Xe.route.id]){Ae=ze[Xe.route.id];break}}if(O.signal.aborted)return{shortCircuited:!0}}if(yo(Ae)){let _e;return Ce&&Ce.replace!=null?_e=Ce.replace:_e=eh(Ae.response.headers.get("Location"),new URL(O.url),f,r.history)===k.location.pathname+k.location.search,await Oe(O,Ae,!0,{submission:Q,replace:_e}),{shortCircuited:!0}}if(gr(Ae)){let _e=qn(ae,Ue.route.id);return(Ce&&Ce.replace)!==!0&&(N="PUSH"),{matches:ae,pendingActionResult:[_e.route.id,Ae,Ue.route.id]}}return{matches:ae,pendingActionResult:[Ue.route.id,Ae]}}async function Dt(O,W,Q,ae,fe,De,Ne,Ce,Se,Ae,Ue,_e,ze){let Xe=De||Uu(W,Ne),_t=Ne||Ce||ih(Xe),At=!oe&&!Ae;if(fe){if(At){let ut=ke(_e);We({navigation:Xe,...ut!==void 0?{actionData:ut}:{}},{flushSync:Ue})}let Qe=await Or(Q,W.pathname,O.signal);if(Qe.type==="aborted")return{shortCircuited:!0};if(Qe.type==="error"){if(Qe.partialMatches.length===0){let{matches:bn,route:Qr}=xs(u);return{matches:bn,loaderData:{},errors:{[Qr.id]:Qe.error}}}let ut=qn(Qe.partialMatches).route.id;return{matches:Qe.partialMatches,loaderData:{},errors:{[ut]:Qe.error}}}else if(Qe.matches)Q=Qe.matches;else{let{error:ut,notFoundMatches:bn,route:Qr}=ur(W.pathname);return{matches:bn,loaderData:{},errors:{[Qr.id]:ut}}}}let Bt=p||u,{dsMatches:nt,revalidatingFetchers:tr}=Xp(O,ae,s,c,r.history,k,Q,_t,W,Ae?[]:a,Ae===!0,re,se,ue,G,$,Bt,f,r.patchRoutesOnNavigation!=null,_e,ze);if(ee=++te,!r.dataStrategy&&!nt.some(Qe=>Qe.shouldLoad)&&!nt.some(Qe=>Qe.route.middleware&&Qe.route.middleware.length>0)&&tr.length===0){let Qe=mn();return Ee(W,{matches:Q,loaderData:{},errors:_e&&gr(_e[1])?{[_e[0]]:_e[1].error}:null,...oh(_e),...Qe?{fetchers:new Map(k.fetchers)}:{}},{flushSync:Ue}),{shortCircuited:!0}}if(At){let Qe={};if(!fe){Qe.navigation=Xe;let ut=ke(_e);ut!==void 0&&(Qe.actionData=ut)}tr.length>0&&(Qe.fetchers=xe(tr)),We(Qe,{flushSync:Ue})}tr.forEach(Qe=>{Yt(Qe.key),Qe.controller&&Y.set(Qe.key,Qe.controller)});let at=()=>tr.forEach(Qe=>Yt(Qe.key));M&&M.signal.addEventListener("abort",at);let{loaderResults:to,fetcherResults:xr}=await rt(nt,tr,O,ae);if(O.signal.aborted)return{shortCircuited:!0};M&&M.signal.removeEventListener("abort",at),tr.forEach(Qe=>Y.delete(Qe.key));let dr=vs(to);if(dr)return await Oe(O,dr.result,!0,{replace:Se}),{shortCircuited:!0};if(dr=vs(xr),dr)return $.add(dr.key),await Oe(O,dr.result,!0,{replace:Se}),{shortCircuited:!0};let{loaderData:Cn,errors:ro}=rh(k,Q,to,_e,tr,xr);Ae&&k.errors&&(ro={...k.errors,...ro});let Mr=mn(),Yr=gn(ee),Qt=Mr||Yr||tr.length>0;return{matches:Q,loaderData:Cn,errors:ro,...Qt?{fetchers:new Map(k.fetchers)}:{}}}function ke(O){if(O&&!gr(O[1]))return{[O[0]]:O[1].data};if(k.actionData)return Object.keys(k.actionData).length===0?null:k.actionData}function xe(O){return O.forEach(W=>{let Q=k.fetchers.get(W.key),ae=za(void 0,Q?Q.data:void 0);k.fetchers.set(W.key,ae)}),new Map(k.fetchers)}async function ye(O,W,Q,ae){Yt(O);let fe=(ae&&ae.flushSync)===!0,De=p||u,Ne=ld(k.location,k.matches,f,Q,W,ae==null?void 0:ae.relative),Ce=Gn(De,Ne,f),Se=Kr(Ce,De,Ne);if(Se.active&&Se.matches&&(Ce=Se.matches),!Ce){mt(O,W,Sr(404,{pathname:Ne}),{flushSync:fe});return}let{path:Ae,submission:Ue,error:_e}=Kp(!0,Ne,ae);if(_e){mt(O,W,_e,{flushSync:fe});return}let ze=r.getContext?await r.getContext():new zp,Xe=(ae&&ae.preventScrollReset)===!0;if(Ue&&Xt(Ue.formMethod)){await Fe(O,W,Ae,Ce,ze,Se.active,fe,Xe,Ue,ae&&ae.unstable_defaultShouldRevalidate);return}G.set(O,{routeId:W,path:Ae}),await He(O,W,Ae,Ce,ze,Se.active,fe,Xe,Ue)}async function Fe(O,W,Q,ae,fe,De,Ne,Ce,Se,Ae){Rt(),G.delete(O);let Ue=k.fetchers.get(O);Ke(O,Y1(Se,Ue),{flushSync:Ne});let _e=new AbortController,ze=ea(r.history,Q,_e.signal,Se);if(De){let vt=await Or(ae,new URL(ze.url).pathname,ze.signal,O);if(vt.type==="aborted")return;if(vt.type==="error"){mt(O,W,vt.error,{flushSync:Ne});return}else if(vt.matches)ae=vt.matches;else{mt(O,W,Sr(404,{pathname:Q}),{flushSync:Ne});return}}let Xe=Ts(ae,Q);if(!Xe.route.action&&!Xe.route.lazy){let vt=Sr(405,{method:Se.formMethod,pathname:Q,routeId:W});mt(O,W,vt,{flushSync:Ne});return}Y.set(O,_e);let _t=te,At=ra(s,c,ze,ae,Xe,a,fe),Bt=await ct(ze,At,fe,O),nt=Bt[Xe.route.id];if(!nt){for(let vt of At)if(Bt[vt.route.id]){nt=Bt[vt.route.id];break}}if(ze.signal.aborted){Y.get(O)===_e&&Y.delete(O);return}if(ue.has(O)){if(yo(nt)||gr(nt)){Ke(O,un(void 0));return}}else{if(yo(nt))if(Y.delete(O),ee>_t){Ke(O,un(void 0));return}else return $.add(O),Ke(O,za(Se)),Oe(ze,nt,!1,{fetcherSubmission:Se,preventScrollReset:Ce});if(gr(nt)){mt(O,W,nt.error);return}}let tr=k.navigation.location||k.location,at=ea(r.history,tr,_e.signal),to=p||u,xr=k.navigation.state!=="idle"?Gn(to,k.navigation.location,f):k.matches;qe(xr,"Didn't find any matches after fetcher action");let dr=++te;_.set(O,dr);let Cn=za(Se,nt.data);k.fetchers.set(O,Cn);let{dsMatches:ro,revalidatingFetchers:Mr}=Xp(at,fe,s,c,r.history,k,xr,Se,tr,a,!1,re,se,ue,G,$,to,f,r.patchRoutesOnNavigation!=null,[Xe.route.id,nt],Ae);Mr.filter(vt=>vt.key!==O).forEach(vt=>{let Jr=vt.key,no=k.fetchers.get(Jr),oo=za(void 0,no?no.data:void 0);k.fetchers.set(Jr,oo),Yt(Jr),vt.controller&&Y.set(Jr,vt.controller)}),We({fetchers:new Map(k.fetchers)});let Yr=()=>Mr.forEach(vt=>Yt(vt.key));_e.signal.addEventListener("abort",Yr);let{loaderResults:Qt,fetcherResults:Qe}=await rt(ro,Mr,at,fe);if(_e.signal.aborted)return;if(_e.signal.removeEventListener("abort",Yr),_.delete(O),Y.delete(O),Mr.forEach(vt=>Y.delete(vt.key)),k.fetchers.has(O)){let vt=un(nt.data);k.fetchers.set(O,vt)}let ut=vs(Qt);if(ut)return Oe(at,ut.result,!1,{preventScrollReset:Ce});if(ut=vs(Qe),ut)return $.add(ut.key),Oe(at,ut.result,!1,{preventScrollReset:Ce});let{loaderData:bn,errors:Qr}=rh(k,xr,Qt,void 0,Mr,Qe);gn(dr),k.navigation.state==="loading"&&dr>ee?(qe(N,"Expected pending action"),M&&M.abort(),Ee(k.navigation.location,{matches:xr,loaderData:bn,errors:Qr,fetchers:new Map(k.fetchers)})):(We({errors:Qr,loaderData:nh(k.loaderData,bn,xr,Qr),fetchers:new Map(k.fetchers)}),re=!1)}async function He(O,W,Q,ae,fe,De,Ne,Ce,Se){let Ae=k.fetchers.get(O);Ke(O,za(Se,Ae?Ae.data:void 0),{flushSync:Ne});let Ue=new AbortController,_e=ea(r.history,Q,Ue.signal);if(De){let nt=await Or(ae,new URL(_e.url).pathname,_e.signal,O);if(nt.type==="aborted")return;if(nt.type==="error"){mt(O,W,nt.error,{flushSync:Ne});return}else if(nt.matches)ae=nt.matches;else{mt(O,W,Sr(404,{pathname:Q}),{flushSync:Ne});return}}let ze=Ts(ae,Q);Y.set(O,Ue);let Xe=te,_t=ra(s,c,_e,ae,ze,a,fe),Bt=(await ct(_e,_t,fe,O))[ze.route.id];if(Y.get(O)===Ue&&Y.delete(O),!_e.signal.aborted){if(ue.has(O)){Ke(O,un(void 0));return}if(yo(Bt))if(ee>Xe){Ke(O,un(void 0));return}else{$.add(O),await Oe(_e,Bt,!1,{preventScrollReset:Ce});return}if(gr(Bt)){mt(O,W,Bt.error);return}Ke(O,un(Bt.data))}}async function Oe(O,W,Q,{submission:ae,fetcherSubmission:fe,preventScrollReset:De,replace:Ne}={}){Q||(I==null||I.resolve(),I=null),W.response.headers.has("X-Remix-Revalidate")&&(re=!0);let Ce=W.response.headers.get("Location");qe(Ce,"Expected a Location header on the redirect Response"),Ce=eh(Ce,new URL(O.url),f,r.history);let Se=ti(k.location,Ce,{_isRedirect:!0});if(n){let _t=!1;if(W.response.headers.has("X-Remix-Reload-Document"))_t=!0;else if(Nl(Ce)){const At=Mm(Ce,!0);_t=At.origin!==t.location.origin||Rr(At.pathname,f)==null}if(_t){Ne?t.location.replace(Ce):t.location.assign(Ce);return}}M=null;let Ae=Ne===!0||W.response.headers.has("X-Remix-Replace")?"REPLACE":"PUSH",{formMethod:Ue,formAction:_e,formEncType:ze}=k.navigation;!ae&&!fe&&Ue&&_e&&ze&&(ae=ih(k.navigation));let Xe=ae||fe;if(R1.has(W.response.status)&&Xe&&Xt(Xe.formMethod))await Ye(Ae,Se,{submission:{...Xe,formAction:Ce},preventScrollReset:De||L,enableViewTransition:Q?H:void 0});else{let _t=Uu(Se,ae);await Ye(Ae,Se,{overrideNavigation:_t,fetcherSubmission:fe,preventScrollReset:De||L,enableViewTransition:Q?H:void 0})}}async function ct(O,W,Q,ae){var Ne;let fe,De={};try{fe=await I1(h,O,W,ae,Q,!1)}catch(Ce){return W.filter(Se=>Se.shouldLoad).forEach(Se=>{De[Se.route.id]={type:"error",error:Ce}}),De}if(O.signal.aborted)return De;if(!Xt(O.method))for(let Ce of W){if(((Ne=fe[Ce.route.id])==null?void 0:Ne.type)==="error")break;!fe.hasOwnProperty(Ce.route.id)&&!k.loaderData.hasOwnProperty(Ce.route.id)&&(!k.errors||!k.errors.hasOwnProperty(Ce.route.id))&&Ce.shouldCallHandler()&&(fe[Ce.route.id]={type:"error",result:new Error(`No result returned from dataStrategy for route ${Ce.route.id}`)})}for(let[Ce,Se]of Object.entries(fe))if(W1(Se)){let Ae=Se.result;De[Ce]={type:"redirect",response:M1(Ae,O,Ce,W,f)}}else De[Ce]=await O1(Se);return De}async function rt(O,W,Q,ae){let fe=ct(Q,O,ae,null),De=Promise.all(W.map(async Se=>{if(Se.matches&&Se.match&&Se.request&&Se.controller){let Ue=(await ct(Se.request,Se.matches,ae,Se.key))[Se.match.route.id];return{[Se.key]:Ue}}else return Promise.resolve({[Se.key]:{type:"error",error:Sr(404,{pathname:Se.path})}})})),Ne=await fe,Ce=(await De).reduce((Se,Ae)=>Object.assign(Se,Ae),{});return{loaderResults:Ne,fetcherResults:Ce}}function Rt(){re=!0,G.forEach((O,W)=>{Y.has(W)&&se.add(W),Yt(W)})}function Ke(O,W,Q={}){k.fetchers.set(O,W),We({fetchers:new Map(k.fetchers)},{flushSync:(Q&&Q.flushSync)===!0})}function mt(O,W,Q,ae={}){let fe=qn(k.matches,W);cr(O),We({errors:{[fe.route.id]:Q},fetchers:new Map(k.fetchers)},{flushSync:(ae&&ae.flushSync)===!0})}function Tt(O){return K.set(O,(K.get(O)||0)+1),ue.has(O)&&ue.delete(O),k.fetchers.get(O)||A1}function Ft(O,W){Yt(O,W==null?void 0:W.reason),Ke(O,un(null))}function cr(O){let W=k.fetchers.get(O);Y.has(O)&&!(W&&W.state==="loading"&&_.has(O))&&Yt(O),G.delete(O),_.delete(O),$.delete(O),ue.delete(O),se.delete(O),k.fetchers.delete(O)}function Po(O){let W=(K.get(O)||0)-1;W<=0?(K.delete(O),ue.add(O)):K.set(O,W),We({fetchers:new Map(k.fetchers)})}function Yt(O,W){let Q=Y.get(O);Q&&(Q.abort(W),Y.delete(O))}function Zn(O){for(let W of O){let Q=Tt(W),ae=un(Q.data);k.fetchers.set(W,ae)}}function mn(){let O=[],W=!1;for(let Q of $){let ae=k.fetchers.get(Q);qe(ae,`Expected fetcher: ${Q}`),ae.state==="loading"&&($.delete(Q),O.push(Q),W=!0)}return Zn(O),W}function gn(O){let W=[];for(let[Q,ae]of _)if(ae<O){let fe=k.fetchers.get(Q);qe(fe,`Expected fetcher: ${Q}`),fe.state==="loading"&&(Yt(Q),_.delete(Q),W.push(Q))}return Zn(W),W.length>0}function xn(O,W){let Q=k.blockers.get(O)||Ua;return me.get(O)!==W&&me.set(O,W),Q}function vn(O){k.blockers.delete(O),me.delete(O)}function Pr(O,W){let Q=k.blockers.get(O)||Ua;qe(Q.state==="unblocked"&&W.state==="blocked"||Q.state==="blocked"&&W.state==="blocked"||Q.state==="blocked"&&W.state==="proceeding"||Q.state==="blocked"&&W.state==="unblocked"||Q.state==="proceeding"&&W.state==="unblocked",`Invalid blocker state transition: ${Q.state} -> ${W.state}`);let ae=new Map(k.blockers);ae.set(O,W),We({blockers:ae})}function kr({currentLocation:O,nextLocation:W,historyAction:Q}){if(me.size===0)return;me.size>1&&bt(!1,"A router only supports one blocker at a time");let ae=Array.from(me.entries()),[fe,De]=ae[ae.length-1],Ne=k.blockers.get(fe);if(!(Ne&&Ne.state==="proceeding")&&De({currentLocation:O,nextLocation:W,historyAction:Q}))return fe}function ur(O){let W=Sr(404,{pathname:O}),Q=p||u,{matches:ae,route:fe}=xs(Q);return{notFoundMatches:ae,route:fe,error:W}}function ko(O,W,Q){if(E=O,S=W,C=Q||null,!P&&k.navigation===$u){P=!0;let ae=yn(k.location,k.matches);ae!=null&&We({restoreScrollPosition:ae})}return()=>{E=null,S=null,C=null}}function eo(O,W){return C&&C(O,W.map(ae=>Jy(ae,k.loaderData)))||O.key}function Do(O,W){if(E&&S){let Q=eo(O,W);E[Q]=S()}}function yn(O,W){if(E){let Q=eo(O,W),ae=E[Q];if(typeof ae=="number")return ae}return null}function Kr(O,W,Q){if(r.patchRoutesOnNavigation)if(O){if(Object.keys(O[0].params).length>0)return{active:!0,matches:qa(W,Q,f,!0)}}else return{active:!0,matches:qa(W,Q,f,!0)||[]};return{active:!1,matches:null}}async function Or(O,W,Q,ae){if(!r.patchRoutesOnNavigation)return{type:"success",matches:O};let fe=O;for(;;){let De=p==null,Ne=p||u,Ce=c;try{await r.patchRoutesOnNavigation({signal:Q,path:W,matches:fe,fetcherKey:ae,patch:(Ue,_e)=>{Q.aborted||Yp(Ue,_e,Ne,Ce,s,!1)}})}catch(Ue){return{type:"error",error:Ue,partialMatches:fe}}finally{De&&!Q.aborted&&(u=[...u])}if(Q.aborted)return{type:"aborted"};let Se=Gn(Ne,W,f),Ae=null;if(Se){if(Object.keys(Se[0].params).length===0)return{type:"success",matches:Se};if(Ae=qa(Ne,W,f,!0),!(Ae&&fe.length<Ae.length&&wn(fe,Ae.slice(0,fe.length))))return{type:"success",matches:Se}}if(Ae||(Ae=qa(Ne,W,f,!0)),!Ae||wn(fe,Ae))return{type:"success",matches:null};fe=Ae}}function wn(O,W){return O.length===W.length&&O.every((Q,ae)=>Q.route.id===W[ae].route.id)}function Xr(O){c={},p=ri(O,s,void 0,c)}function En(O,W,Q=!1){let ae=p==null;Yp(O,W,p||u,c,s,Q),ae&&(u=[...u],We({}))}return F={get basename(){return f},get future(){return x},get state(){return k},get routes(){return u},get window(){return t},initialize:Ve,subscribe:Vt,enableScrollRestoration:ko,navigate:ge,fetch:ye,revalidate:je,createHref:O=>r.history.createHref(O),encodeLocation:O=>r.history.encodeLocation(O),getFetcher:Tt,resetFetcher:Ft,deleteFetcher:Po,dispose:ot,getBlocker:xn,deleteBlocker:vn,patchRoutes:En,_internalFetchControllers:Y,_internalSetRoutes:Xr,_internalSetStateDoNotUseOrYouWillBreakYourApp(O){We(O)}},r.unstable_instrumentations&&(F=x1(F,r.unstable_instrumentations.map(O=>O.router).filter(Boolean))),F}function D1(r){return r!=null&&("formData"in r&&r.formData!=null||"body"in r&&r.body!==void 0)}function ld(r,t,n,a,i,s){let c,u;if(i){c=[];for(let f of t)if(c.push(f),f.route.id===i){u=f;break}}else c=t,u=t[t.length-1];let p=wd(a||".",yd(c),Rr(r.pathname,n)||r.pathname,s==="path");if(a==null&&(p.search=r.search,p.hash=r.hash),(a==null||a===""||a===".")&&u){let f=bd(p.search);if(u.route.index&&!f)p.search=p.search?p.search.replace(/^\?/,"?index&"):"?index";else if(!u.route.index&&f){let h=new URLSearchParams(p.search),x=h.getAll("index");h.delete("index"),x.filter(w=>w).forEach(w=>h.append("index",w));let y=h.toString();p.search=y?`?${y}`:""}}return n!=="/"&&(p.pathname=d1({basename:n,pathname:p.pathname})),qr(p)}function Kp(r,t,n){if(!n||!D1(n))return{path:t};if(n.formMethod&&!K1(n.formMethod))return{path:t,error:Sr(405,{method:n.formMethod})};let a=()=>({path:t,error:Sr(400,{type:"invalid-body"})}),s=(n.formMethod||"get").toUpperCase(),c=rg(t);if(n.body!==void 0){if(n.formEncType==="text/plain"){if(!Xt(s))return a();let x=typeof n.body=="string"?n.body:n.body instanceof FormData||n.body instanceof URLSearchParams?Array.from(n.body.entries()).reduce((y,[w,E])=>`${y}${w}=${E}
`,""):String(n.body);return{path:t,submission:{formMethod:s,formAction:c,formEncType:n.formEncType,formData:void 0,json:void 0,text:x}}}else if(n.formEncType==="application/json"){if(!Xt(s))return a();try{let x=typeof n.body=="string"?JSON.parse(n.body):n.body;return{path:t,submission:{formMethod:s,formAction:c,formEncType:n.formEncType,formData:void 0,json:x,text:void 0}}}catch{return a()}}}qe(typeof FormData=="function","FormData is not available in this environment");let u,p;if(n.formData)u=dd(n.formData),p=n.formData;else if(n.body instanceof FormData)u=dd(n.body),p=n.body;else if(n.body instanceof URLSearchParams)u=n.body,p=th(u);else if(n.body==null)u=new URLSearchParams,p=new FormData;else try{u=new URLSearchParams(n.body),p=th(u)}catch{return a()}let f={formMethod:s,formAction:c,formEncType:n&&n.formEncType||"application/x-www-form-urlencoded",formData:p,json:void 0,text:void 0};if(Xt(f.formMethod))return{path:t,submission:f};let h=fn(t);return r&&h.search&&bd(h.search)&&u.append("index",""),h.search=`?${u}`,{path:qr(h),submission:f}}function Xp(r,t,n,a,i,s,c,u,p,f,h,x,y,w,E,C,S,P,b,A,T){var oe;let B=A?gr(A[1])?A[1].error:A[1].data:void 0,F=i.createURL(s.location),k=i.createURL(p),N;if(h&&s.errors){let re=Object.keys(s.errors)[0];N=c.findIndex(se=>se.route.id===re)}else if(A&&gr(A[1])){let re=A[0];N=c.findIndex(se=>se.route.id===re)-1}let I=A?A[1].statusCode:void 0,L=I&&I>=400,M={currentUrl:F,currentParams:((oe=s.matches[0])==null?void 0:oe.params)||{},nextUrl:k,nextParams:c[0].params,...u,actionResult:B,actionStatus:I},H=ci(c),z=c.map((re,se)=>{let{route:Y}=re,te=null;if(N!=null&&se>N?te=!1:Y.lazy?te=!0:Ed(Y)?h?te=cd(Y,s.loaderData,s.errors):T1(s.loaderData,s.matches[se],re)&&(te=!0):te=!1,te!==null)return ud(n,a,r,H,re,f,t,te);let ee=!1;typeof T=="boolean"?ee=T:L?ee=!1:(x||F.pathname+F.search===k.pathname+k.search||F.search!==k.search||_1(s.matches[se],re))&&(ee=!0);let _={...M,defaultShouldRevalidate:ee},$=Za(re,_);return ud(n,a,r,H,re,f,t,$,_,T)}),Z=[];return E.forEach((re,se)=>{if(h||!c.some(ue=>ue.route.id===re.routeId)||w.has(se))return;let Y=s.fetchers.get(se),te=Y&&Y.state!=="idle"&&Y.data===void 0,ee=Gn(S,re.path,P);if(!ee){if(b&&te)return;Z.push({key:se,routeId:re.routeId,path:re.path,matches:null,match:null,request:null,controller:null});return}if(C.has(se))return;let _=Ts(ee,re.path),$=new AbortController,G=ea(i,re.path,$.signal),K=null;if(y.has(se))y.delete(se),K=ra(n,a,G,ee,_,f,t);else if(te)x&&(K=ra(n,a,G,ee,_,f,t));else{let ue;typeof T=="boolean"?ue=T:L?ue=!1:ue=x;let me={...M,defaultShouldRevalidate:ue};Za(_,me)&&(K=ra(n,a,G,ee,_,f,t,me))}K&&Z.push({key:se,routeId:re.routeId,path:re.path,matches:K,match:_,request:G,controller:$})}),{dsMatches:z,revalidatingFetchers:Z}}function Ed(r){return r.loader!=null||r.middleware!=null&&r.middleware.length>0}function cd(r,t,n){if(r.lazy)return!0;if(!Ed(r))return!1;let a=t!=null&&r.id in t,i=n!=null&&n[r.id]!==void 0;return!a&&i?!1:typeof r.loader=="function"&&r.loader.hydrate===!0?!0:!a&&!i}function T1(r,t,n){let a=!t||n.route.id!==t.route.id,i=!r.hasOwnProperty(n.route.id);return a||i}function _1(r,t){let n=r.route.path;return r.pathname!==t.pathname||n!=null&&n.endsWith("*")&&r.params["*"]!==t.params["*"]}function Za(r,t){if(r.route.shouldRevalidate){let n=r.route.shouldRevalidate(t);if(typeof n=="boolean")return n}return t.defaultShouldRevalidate}function Yp(r,t,n,a,i,s){let c;if(r){let f=a[r];qe(f,`No route found to patch children into: routeId = ${r}`),f.children||(f.children=[]),c=f.children}else c=n;let u=[],p=[];if(t.forEach(f=>{let h=c.find(x=>Qm(f,x));h?p.push({existingRoute:h,newRoute:f}):u.push(f)}),u.length>0){let f=ri(u,i,[r||"_","patch",String((c==null?void 0:c.length)||"0")],a);c.push(...f)}if(s&&p.length>0)for(let f=0;f<p.length;f++){let{existingRoute:h,newRoute:x}=p[f],y=h,[w]=ri([x],i,[],{},!0);Object.assign(y,{element:w.element?w.element:y.element,errorElement:w.errorElement?w.errorElement:y.errorElement,hydrateFallbackElement:w.hydrateFallbackElement?w.hydrateFallbackElement:y.hydrateFallbackElement})}}function Qm(r,t){return"id"in r&&"id"in t&&r.id===t.id?!0:r.index===t.index&&r.path===t.path&&r.caseSensitive===t.caseSensitive?(!r.children||r.children.length===0)&&(!t.children||t.children.length===0)?!0:r.children.every((n,a)=>{var i;return(i=t.children)==null?void 0:i.some(s=>Qm(n,s))}):!1}var Qp=new WeakMap,Jm=({key:r,route:t,manifest:n,mapRouteProperties:a})=>{let i=n[t.id];if(qe(i,"No route found in manifest"),!i.lazy||typeof i.lazy!="object")return;let s=i.lazy[r];if(!s)return;let c=Qp.get(i);c||(c={},Qp.set(i,c));let u=c[r];if(u)return u;let p=(async()=>{let f=Ky(r),x=i[r]!==void 0&&r!=="hasErrorBoundary";if(f)bt(!f,"Route property "+r+" is not a supported lazy route property. This property will be ignored."),c[r]=Promise.resolve();else if(x)bt(!1,`Route "${i.id}" has a static property "${r}" defined. The lazy property will be ignored.`);else{let y=await s();y!=null&&(Object.assign(i,{[r]:y}),Object.assign(i,a(i)))}typeof i.lazy=="object"&&(i.lazy[r]=void 0,Object.values(i.lazy).every(y=>y===void 0)&&(i.lazy=void 0))})();return c[r]=p,p},Jp=new WeakMap;function L1(r,t,n,a,i){let s=n[r.id];if(qe(s,"No route found in manifest"),!r.lazy)return{lazyRoutePromise:void 0,lazyHandlerPromise:void 0};if(typeof r.lazy=="function"){let h=Jp.get(s);if(h)return{lazyRoutePromise:h,lazyHandlerPromise:h};let x=(async()=>{qe(typeof r.lazy=="function","No lazy route function found");let y=await r.lazy(),w={};for(let E in y){let C=y[E];if(C===void 0)continue;let S=Yy(E),b=s[E]!==void 0&&E!=="hasErrorBoundary";S?bt(!S,"Route property "+E+" is not a supported property to be returned from a lazy route function. This property will be ignored."):b?bt(!b,`Route "${s.id}" has a static property "${E}" defined but its lazy function is also returning a value for this property. The lazy route property "${E}" will be ignored.`):w[E]=C}Object.assign(s,w),Object.assign(s,{...a(s),lazy:void 0})})();return Jp.set(s,x),x.catch(()=>{}),{lazyRoutePromise:x,lazyHandlerPromise:x}}let c=Object.keys(r.lazy),u=[],p;for(let h of c){if(i&&i.includes(h))continue;let x=Jm({key:h,route:r,manifest:n,mapRouteProperties:a});x&&(u.push(x),h===t&&(p=x))}let f=u.length>0?Promise.all(u).then(()=>{}):void 0;return f==null||f.catch(()=>{}),p==null||p.catch(()=>{}),{lazyRoutePromise:f,lazyHandlerPromise:p}}async function Zp(r){let t=r.matches.filter(i=>i.shouldLoad),n={};return(await Promise.all(t.map(i=>i.resolve()))).forEach((i,s)=>{n[t[s].route.id]=i}),n}async function F1(r){return r.matches.some(t=>t.route.middleware)?Zm(r,()=>Zp(r)):Zp(r)}function Zm(r,t){return N1(r,t,a=>{if(q1(a))throw a;return a},H1,n);function n(a,i,s){if(s)return Promise.resolve(Object.assign(s.value,{[i]:{type:"error",result:a}}));{let{matches:c}=r,u=Math.min(Math.max(c.findIndex(f=>f.route.id===i),0),Math.max(c.findIndex(f=>f.shouldCallHandler()),0)),p=qn(c,c[u].route.id).route.id;return Promise.resolve({[p]:{type:"error",result:a}})}}}async function N1(r,t,n,a,i){let{matches:s,request:c,params:u,context:p,unstable_pattern:f}=r,h=s.flatMap(y=>y.route.middleware?y.route.middleware.map(w=>[y.route.id,w]):[]);return await eg({request:c,params:u,context:p,unstable_pattern:f},h,t,n,a,i)}async function eg(r,t,n,a,i,s,c=0){let{request:u}=r;if(u.signal.aborted)throw u.signal.reason??new Error(`Request aborted: ${u.method} ${u.url}`);let p=t[c];if(!p)return await n();let[f,h]=p,x,y=async()=>{if(x)throw new Error("You may only call `next()` once per middleware");try{return x={value:await eg(r,t,n,a,i,s,c+1)},x.value}catch(w){return x={value:await s(w,f,x)},x.value}};try{let w=await h(r,y),E=w!=null?a(w):void 0;return i(E)?E:x?E??x.value:(x={value:await y()},x.value)}catch(w){return await s(w,f,x)}}function tg(r,t,n,a,i){let s=Jm({key:"middleware",route:a.route,manifest:t,mapRouteProperties:r}),c=L1(a.route,Xt(n.method)?"action":"loader",t,r,i);return{middleware:s,route:c.lazyRoutePromise,handler:c.lazyHandlerPromise}}function ud(r,t,n,a,i,s,c,u,p=null,f){let h=!1,x=tg(r,t,n,i,s);return{...i,_lazyPromises:x,shouldLoad:u,shouldRevalidateArgs:p,shouldCallHandler(y){return h=!0,p?typeof f=="boolean"?Za(i,{...p,defaultShouldRevalidate:f}):typeof y=="boolean"?Za(i,{...p,defaultShouldRevalidate:y}):Za(i,p):u},resolve(y){let{lazy:w,loader:E,middleware:C}=i.route,S=h||u||y&&!Xt(n.method)&&(w||E),P=C&&C.length>0&&!E&&!w;return S&&(Xt(n.method)||!P)?B1({request:n,unstable_pattern:a,match:i,lazyHandlerPromise:x==null?void 0:x.handler,lazyRoutePromise:x==null?void 0:x.route,handlerOverride:y,scopedContext:c}):Promise.resolve({type:"data",result:void 0})}}}function ra(r,t,n,a,i,s,c,u=null){return a.map(p=>p.route.id!==i.route.id?{...p,shouldLoad:!1,shouldRevalidateArgs:u,shouldCallHandler:()=>!1,_lazyPromises:tg(r,t,n,p,s),resolve:()=>Promise.resolve({type:"data",result:void 0})}:ud(r,t,n,ci(a),p,s,c,!0,u))}async function I1(r,t,n,a,i,s){n.some(f=>{var h;return(h=f._lazyPromises)==null?void 0:h.middleware})&&await Promise.all(n.map(f=>{var h;return(h=f._lazyPromises)==null?void 0:h.middleware}));let c={request:t,unstable_pattern:ci(n),params:n[0].params,context:i,matches:n},p=await r({...c,fetcherKey:a,runClientMiddleware:f=>{let h=c;return Zm(h,()=>f({...h,fetcherKey:a,runClientMiddleware:()=>{throw new Error("Cannot call `runClientMiddleware()` from within an `runClientMiddleware` handler")}}))}});try{await Promise.all(n.flatMap(f=>{var h,x;return[(h=f._lazyPromises)==null?void 0:h.handler,(x=f._lazyPromises)==null?void 0:x.route]}))}catch{}return p}async function B1({request:r,unstable_pattern:t,match:n,lazyHandlerPromise:a,lazyRoutePromise:i,handlerOverride:s,scopedContext:c}){let u,p,f=Xt(r.method),h=f?"action":"loader",x=y=>{let w,E=new Promise((P,b)=>w=b);p=()=>w(),r.signal.addEventListener("abort",p);let C=P=>typeof y!="function"?Promise.reject(new Error(`You cannot call the handler for a route which defines a boolean "${h}" [routeId: ${n.route.id}]`)):y({request:r,unstable_pattern:t,params:n.params,context:c},...P!==void 0?[P]:[]),S=(async()=>{try{return{type:"data",result:await(s?s(b=>C(b)):C())}}catch(P){return{type:"error",result:P}}})();return Promise.race([S,E])};try{let y=f?n.route.action:n.route.loader;if(a||i)if(y){let w,[E]=await Promise.all([x(y).catch(C=>{w=C}),a,i]);if(w!==void 0)throw w;u=E}else{await a;let w=f?n.route.action:n.route.loader;if(w)[u]=await Promise.all([x(w),i]);else if(h==="action"){let E=new URL(r.url),C=E.pathname+E.search;throw Sr(405,{method:r.method,pathname:C,routeId:n.route.id})}else return{type:"data",result:void 0}}else if(y)u=await x(y);else{let w=new URL(r.url),E=w.pathname+w.search;throw Sr(404,{pathname:E})}}catch(y){return{type:"error",result:y}}finally{p&&r.signal.removeEventListener("abort",p)}return u}async function j1(r){let t=r.headers.get("Content-Type");return t&&/\bapplication\/json\b/.test(t)?r.body==null?null:r.json():r.text()}async function O1(r){var a,i,s,c,u;let{result:t,type:n}=r;if(Cd(t)){let p;try{p=await j1(t)}catch(f){return{type:"error",error:f}}return n==="error"?{type:"error",error:new li(t.status,t.statusText,p),statusCode:t.status,headers:t.headers}:{type:"data",data:p,statusCode:t.status,headers:t.headers}}return n==="error"?ah(t)?t.data instanceof Error?{type:"error",error:t.data,statusCode:(a=t.init)==null?void 0:a.status,headers:(i=t.init)!=null&&i.headers?new Headers(t.init.headers):void 0}:{type:"error",error:z1(t),statusCode:ni(t)?t.status:void 0,headers:(s=t.init)!=null&&s.headers?new Headers(t.init.headers):void 0}:{type:"error",error:t,statusCode:ni(t)?t.status:void 0}:ah(t)?{type:"data",data:t.data,statusCode:(c=t.init)==null?void 0:c.status,headers:(u=t.init)!=null&&u.headers?new Headers(t.init.headers):void 0}:{type:"data",data:t}}function M1(r,t,n,a,i){let s=r.headers.get("Location");if(qe(s,"Redirects returned/thrown from loaders/actions must have a Location header"),!Nl(s)){let c=a.slice(0,a.findIndex(u=>u.route.id===n)+1);s=ld(new URL(t.url),c,i,s),r.headers.set("Location",s)}return r}function eh(r,t,n,a){let i=["about:","blob:","chrome:","chrome-untrusted:","content:","data:","devtools:","file:","filesystem:","javascript:"];if(Nl(r)){let s=r,c=s.startsWith("//")?new URL(t.protocol+s):new URL(s);if(i.includes(c.protocol))throw new Error("Invalid redirect location");let u=Rr(c.pathname,n)!=null;if(c.origin===t.origin&&u)return c.pathname+c.search+c.hash}try{let s=a.createURL(r);if(i.includes(s.protocol))throw new Error("Invalid redirect location")}catch{}return r}function ea(r,t,n,a){let i=r.createURL(rg(t)).toString(),s={signal:n};if(a&&Xt(a.formMethod)){let{formMethod:c,formEncType:u}=a;s.method=c.toUpperCase(),u==="application/json"?(s.headers=new Headers({"Content-Type":u}),s.body=JSON.stringify(a.json)):u==="text/plain"?s.body=a.text:u==="application/x-www-form-urlencoded"&&a.formData?s.body=dd(a.formData):s.body=a.formData}return new Request(i,s)}function dd(r){let t=new URLSearchParams;for(let[n,a]of r.entries())t.append(n,typeof a=="string"?a:a.name);return t}function th(r){let t=new FormData;for(let[n,a]of r.entries())t.append(n,a);return t}function $1(r,t,n,a=!1,i=!1){let s={},c=null,u,p=!1,f={},h=n&&gr(n[1])?n[1].error:void 0;return r.forEach(x=>{if(!(x.route.id in t))return;let y=x.route.id,w=t[y];if(qe(!yo(w),"Cannot handle redirect results in processLoaderData"),gr(w)){let E=w.error;if(h!==void 0&&(E=h,h=void 0),c=c||{},i)c[y]=E;else{let C=qn(r,y);c[C.route.id]==null&&(c[C.route.id]=E)}a||(s[y]=Ym),p||(p=!0,u=ni(w.error)?w.error.status:500),w.headers&&(f[y]=w.headers)}else s[y]=w.data,w.statusCode&&w.statusCode!==200&&!p&&(u=w.statusCode),w.headers&&(f[y]=w.headers)}),h!==void 0&&n&&(c={[n[0]]:h},n[2]&&(s[n[2]]=void 0)),{loaderData:s,errors:c,statusCode:u||200,loaderHeaders:f}}function rh(r,t,n,a,i,s){let{loaderData:c,errors:u}=$1(t,n,a);return i.filter(p=>!p.matches||p.matches.some(f=>f.shouldLoad)).forEach(p=>{let{key:f,match:h,controller:x}=p;if(x&&x.signal.aborted)return;let y=s[f];if(qe(y,"Did not find corresponding fetcher result"),gr(y)){let w=qn(r.matches,h==null?void 0:h.route.id);u&&u[w.route.id]||(u={...u,[w.route.id]:y.error}),r.fetchers.delete(f)}else if(yo(y))qe(!1,"Unhandled fetcher revalidation redirect");else{let w=un(y.data);r.fetchers.set(f,w)}}),{loaderData:c,errors:u}}function nh(r,t,n,a){let i=Object.entries(t).filter(([,s])=>s!==Ym).reduce((s,[c,u])=>(s[c]=u,s),{});for(let s of n){let c=s.route.id;if(!t.hasOwnProperty(c)&&r.hasOwnProperty(c)&&s.route.loader&&(i[c]=r[c]),a&&a.hasOwnProperty(c))break}return i}function oh(r){return r?gr(r[1])?{actionData:{}}:{actionData:{[r[0]]:r[1].data}}:{}}function qn(r,t){return(t?r.slice(0,r.findIndex(a=>a.route.id===t)+1):[...r]).reverse().find(a=>a.route.hasErrorBoundary===!0)||r[0]}function xs(r){let t=r.length===1?r[0]:r.find(n=>n.index||!n.path||n.path==="/")||{id:"__shim-error-route__"};return{matches:[{params:{},pathname:"",pathnameBase:"",route:t}],route:t}}function Sr(r,{pathname:t,routeId:n,method:a,type:i,message:s}={}){let c="Unknown Server Error",u="Unknown @remix-run/router error";return r===400?(c="Bad Request",a&&t&&n?u=`You made a ${a} request to "${t}" but did not provide a \`loader\` for route "${n}", so there is no way to handle the request.`:i==="invalid-body"&&(u="Unable to encode submission body")):r===403?(c="Forbidden",u=`Route "${n}" does not match URL "${t}"`):r===404?(c="Not Found",u=`No route matches URL "${t}"`):r===405&&(c="Method Not Allowed",a&&t&&n?u=`You made a ${a.toUpperCase()} request to "${t}" but did not provide an \`action\` for route "${n}", so there is no way to handle the request.`:a&&(u=`Invalid request method "${a.toUpperCase()}"`)),new li(r||500,c,new Error(u),!0)}function vs(r){let t=Object.entries(r);for(let n=t.length-1;n>=0;n--){let[a,i]=t[n];if(yo(i))return{key:a,result:i}}}function rg(r){let t=typeof r=="string"?fn(r):r;return qr({...t,hash:""})}function U1(r,t){return r.pathname!==t.pathname||r.search!==t.search?!1:r.hash===""?t.hash!=="":r.hash===t.hash?!0:t.hash!==""}function z1(r){var t,n;return new li(((t=r.init)==null?void 0:t.status)??500,((n=r.init)==null?void 0:n.statusText)??"Internal Server Error",r.data)}function H1(r){return r!=null&&typeof r=="object"&&Object.entries(r).every(([t,n])=>typeof t=="string"&&V1(n))}function V1(r){return r!=null&&typeof r=="object"&&"type"in r&&"result"in r&&(r.type==="data"||r.type==="error")}function W1(r){return Cd(r.result)&&Km.has(r.result.status)}function gr(r){return r.type==="error"}function yo(r){return(r&&r.type)==="redirect"}function ah(r){return typeof r=="object"&&r!=null&&"type"in r&&"data"in r&&"init"in r&&r.type==="DataWithResponseInit"}function Cd(r){return r!=null&&typeof r.status=="number"&&typeof r.statusText=="string"&&typeof r.headers=="object"&&typeof r.body<"u"}function G1(r){return Km.has(r)}function q1(r){return Cd(r)&&G1(r.status)&&r.headers.has("Location")}function K1(r){return S1.has(r.toUpperCase())}function Xt(r){return C1.has(r.toUpperCase())}function bd(r){return new URLSearchParams(r).getAll("index").some(t=>t==="")}function Ts(r,t){let n=typeof t=="string"?fn(t).search:t.search;if(r[r.length-1].route.index&&bd(n||""))return r[r.length-1];let a=Hm(r);return a[a.length-1]}function ih(r){let{formMethod:t,formAction:n,formEncType:a,text:i,formData:s,json:c}=r;if(!(!t||!n||!a)){if(i!=null)return{formMethod:t,formAction:n,formEncType:a,formData:void 0,json:void 0,text:i};if(s!=null)return{formMethod:t,formAction:n,formEncType:a,formData:s,json:void 0,text:void 0};if(c!==void 0)return{formMethod:t,formAction:n,formEncType:a,formData:void 0,json:c,text:void 0}}}function Uu(r,t){return t?{state:"loading",location:r,formMethod:t.formMethod,formAction:t.formAction,formEncType:t.formEncType,formData:t.formData,json:t.json,text:t.text}:{state:"loading",location:r,formMethod:void 0,formAction:void 0,formEncType:void 0,formData:void 0,json:void 0,text:void 0}}function X1(r,t){return{state:"submitting",location:r,formMethod:t.formMethod,formAction:t.formAction,formEncType:t.formEncType,formData:t.formData,json:t.json,text:t.text}}function za(r,t){return r?{state:"loading",formMethod:r.formMethod,formAction:r.formAction,formEncType:r.formEncType,formData:r.formData,json:r.json,text:r.text,data:t}:{state:"loading",formMethod:void 0,formAction:void 0,formEncType:void 0,formData:void 0,json:void 0,text:void 0,data:t}}function Y1(r,t){return{state:"submitting",formMethod:r.formMethod,formAction:r.formAction,formEncType:r.formEncType,formData:r.formData,json:r.json,text:r.text,data:t?t.data:void 0}}function un(r){return{state:"idle",formMethod:void 0,formAction:void 0,formEncType:void 0,formData:void 0,json:void 0,text:void 0,data:r}}function Q1(r,t){try{let n=r.sessionStorage.getItem(Xm);if(n){let a=JSON.parse(n);for(let[i,s]of Object.entries(a||{}))s&&Array.isArray(s)&&t.set(i,new Set(s||[]))}}catch{}}function J1(r,t){if(t.size>0){let n={};for(let[a,i]of t)n[a]=[...i];try{r.sessionStorage.setItem(Xm,JSON.stringify(n))}catch(a){bt(!1,`Failed to save applied view transitions in sessionStorage (${a}).`)}}}function sh(){let r,t,n=new Promise((a,i)=>{r=async s=>{a(s);try{await n}catch{}},t=async s=>{i(s);try{await n}catch{}}});return{promise:n,resolve:r,reject:t}}var So=D.createContext(null);So.displayName="DataRouter";var ui=D.createContext(null);ui.displayName="DataRouterState";var ng=D.createContext(!1);function Z1(){return D.useContext(ng)}var Sd=D.createContext({isTransitioning:!1});Sd.displayName="ViewTransition";var og=D.createContext(new Map);og.displayName="Fetchers";var ew=D.createContext(null);ew.displayName="Await";var Ar=D.createContext(null);Ar.displayName="Navigation";var Il=D.createContext(null);Il.displayName="Location";var pn=D.createContext({outlet:null,matches:[],isDataRoute:!1});pn.displayName="Route";var Rd=D.createContext(null);Rd.displayName="RouteError";var ag="REACT_ROUTER_ERROR",tw="REDIRECT",rw="ROUTE_ERROR_RESPONSE";function nw(r){if(r.startsWith(`${ag}:${tw}:{`))try{let t=JSON.parse(r.slice(28));if(typeof t=="object"&&t&&typeof t.status=="number"&&typeof t.statusText=="string"&&typeof t.location=="string"&&typeof t.reloadDocument=="boolean"&&typeof t.replace=="boolean")return t}catch{}}function ow(r){if(r.startsWith(`${ag}:${rw}:{`))try{let t=JSON.parse(r.slice(40));if(typeof t=="object"&&t&&typeof t.status=="number"&&typeof t.statusText=="string")return new li(t.status,t.statusText,t.data)}catch{}}function aw(r,{relative:t}={}){qe(di(),"useHref() may be used only in the context of a <Router> component.");let{basename:n,navigator:a}=D.useContext(Ar),{hash:i,pathname:s,search:c}=fi(r,{relative:t}),u=s;return n!=="/"&&(u=s==="/"?n:Gr([n,s])),a.createHref({pathname:u,search:c,hash:i})}function di(){return D.useContext(Il)!=null}function hn(){return qe(di(),"useLocation() may be used only in the context of a <Router> component."),D.useContext(Il).location}var ig="You should call navigate() in a React.useEffect(), not when your component is first rendered.";function sg(r){D.useContext(Ar).static||D.useLayoutEffect(r)}function lg(){let{isDataRoute:r}=D.useContext(pn);return r?vw():iw()}function iw(){qe(di(),"useNavigate() may be used only in the context of a <Router> component.");let r=D.useContext(So),{basename:t,navigator:n}=D.useContext(Ar),{matches:a}=D.useContext(pn),{pathname:i}=hn(),s=JSON.stringify(yd(a)),c=D.useRef(!1);return sg(()=>{c.current=!0}),D.useCallback((p,f={})=>{if(bt(c.current,ig),!c.current)return;if(typeof p=="number"){n.go(p);return}let h=wd(p,JSON.parse(s),i,f.relative==="path");r==null&&t!=="/"&&(h.pathname=h.pathname==="/"?t:Gr([t,h.pathname])),(f.replace?n.replace:n.push)(h,f.state,f)},[t,n,s,i,r])}D.createContext(null);function fi(r,{relative:t}={}){let{matches:n}=D.useContext(pn),{pathname:a}=hn(),i=JSON.stringify(yd(n));return D.useMemo(()=>wd(r,JSON.parse(i),a,t==="path"),[r,i,a,t])}function sw(r,t,n,a,i){qe(di(),"useRoutes() may be used only in the context of a <Router> component.");let{navigator:s}=D.useContext(Ar),{matches:c}=D.useContext(pn),u=c[c.length-1],p=u?u.params:{},f=u?u.pathname:"/",h=u?u.pathnameBase:"/",x=u&&u.route;{let b=x&&x.path||"";ug(f,!x||b.endsWith("*")||b.endsWith("*?"),`You rendered descendant <Routes> (or called \`useRoutes()\`) at "${f}" (under <Route path="${b}">) but the parent route path has no trailing "*". This means if you navigate deeper, the parent won't match anymore and therefore the child routes will never render.

Please change the parent <Route path="${b}"> to <Route path="${b==="/"?"*":`${b}/*`}">.`)}let y=hn(),w;w=y;let E=w.pathname||"/",C=E;if(h!=="/"){let b=h.replace(/^\//,"").split("/");C="/"+E.replace(/^\//,"").split("/").slice(b.length).join("/")}let S=Gn(r,{pathname:C});return bt(x||S!=null,`No routes matched location "${w.pathname}${w.search}${w.hash}" `),bt(S==null||S[S.length-1].route.element!==void 0||S[S.length-1].route.Component!==void 0||S[S.length-1].route.lazy!==void 0,`Matched leaf route at location "${w.pathname}${w.search}${w.hash}" does not have an element or Component. This means it will render an <Outlet /> with a null value by default resulting in an "empty" page.`),fw(S&&S.map(b=>Object.assign({},b,{params:Object.assign({},p,b.params),pathname:Gr([h,s.encodeLocation?s.encodeLocation(b.pathname.replace(/\?/g,"%3F").replace(/#/g,"%23")).pathname:b.pathname]),pathnameBase:b.pathnameBase==="/"?h:Gr([h,s.encodeLocation?s.encodeLocation(b.pathnameBase.replace(/\?/g,"%3F").replace(/#/g,"%23")).pathname:b.pathnameBase])})),c,n,a,i)}function lw(){let r=xw(),t=ni(r)?`${r.status} ${r.statusText}`:r instanceof Error?r.message:JSON.stringify(r),n=r instanceof Error?r.stack:null,a="rgba(200,200,200, 0.5)",i={padding:"0.5rem",backgroundColor:a},s={padding:"2px 4px",backgroundColor:a},c=null;return console.error("Error handled by React Router default ErrorBoundary:",r),c=D.createElement(D.Fragment,null,D.createElement("p",null,"💿 Hey developer 👋"),D.createElement("p",null,"You can provide a way better UX than this when your app throws errors by providing your own ",D.createElement("code",{style:s},"ErrorBoundary")," or"," ",D.createElement("code",{style:s},"errorElement")," prop on your route.")),D.createElement(D.Fragment,null,D.createElement("h2",null,"Unexpected Application Error!"),D.createElement("h3",{style:{fontStyle:"italic"}},t),n?D.createElement("pre",{style:i},n):null,c)}var cw=D.createElement(lw,null),cg=class extends D.Component{constructor(r){super(r),this.state={location:r.location,revalidation:r.revalidation,error:r.error}}static getDerivedStateFromError(r){return{error:r}}static getDerivedStateFromProps(r,t){return t.location!==r.location||t.revalidation!=="idle"&&r.revalidation==="idle"?{error:r.error,location:r.location,revalidation:r.revalidation}:{error:r.error!==void 0?r.error:t.error,location:t.location,revalidation:r.revalidation||t.revalidation}}componentDidCatch(r,t){this.props.onError?this.props.onError(r,t):console.error("React Router caught the following error during render",r)}render(){let r=this.state.error;if(this.context&&typeof r=="object"&&r&&"digest"in r&&typeof r.digest=="string"){const n=ow(r.digest);n&&(r=n)}let t=r!==void 0?D.createElement(pn.Provider,{value:this.props.routeContext},D.createElement(Rd.Provider,{value:r,children:this.props.component})):this.props.children;return this.context?D.createElement(uw,{error:r},t):t}};cg.contextType=ng;var zu=new WeakMap;function uw({children:r,error:t}){let{basename:n}=D.useContext(Ar);if(typeof t=="object"&&t&&"digest"in t&&typeof t.digest=="string"){let a=nw(t.digest);if(a){let i=zu.get(t);if(i)throw i;let s=Wm(a.location,n);if(Vm&&!zu.get(t))if(s.isExternal||a.reloadDocument)window.location.href=s.absoluteURL||s.to;else{const c=Promise.resolve().then(()=>window.__reactRouterDataRouter.navigate(s.to,{replace:a.replace}));throw zu.set(t,c),c}return D.createElement("meta",{httpEquiv:"refresh",content:`0;url=${s.absoluteURL||s.to}`})}}return r}function dw({routeContext:r,match:t,children:n}){let a=D.useContext(So);return a&&a.static&&a.staticContext&&(t.route.errorElement||t.route.ErrorBoundary)&&(a.staticContext._deepestRenderedBoundaryId=t.route.id),D.createElement(pn.Provider,{value:r},n)}function fw(r,t=[],n=null,a=null,i=null){if(r==null){if(!n)return null;if(n.errors)r=n.matches;else if(t.length===0&&!n.initialized&&n.matches.length>0)r=n.matches;else return null}let s=r,c=n==null?void 0:n.errors;if(c!=null){let h=s.findIndex(x=>x.route.id&&(c==null?void 0:c[x.route.id])!==void 0);qe(h>=0,`Could not find a matching route for errors on route IDs: ${Object.keys(c).join(",")}`),s=s.slice(0,Math.min(s.length,h+1))}let u=!1,p=-1;if(n)for(let h=0;h<s.length;h++){let x=s[h];if((x.route.HydrateFallback||x.route.hydrateFallbackElement)&&(p=h),x.route.id){let{loaderData:y,errors:w}=n,E=x.route.loader&&!y.hasOwnProperty(x.route.id)&&(!w||w[x.route.id]===void 0);if(x.route.lazy||E){u=!0,p>=0?s=s.slice(0,p+1):s=[s[0]];break}}}let f=n&&a?(h,x)=>{var y,w;a(h,{location:n.location,params:((w=(y=n.matches)==null?void 0:y[0])==null?void 0:w.params)??{},unstable_pattern:ci(n.matches),errorInfo:x})}:void 0;return s.reduceRight((h,x,y)=>{let w,E=!1,C=null,S=null;n&&(w=c&&x.route.id?c[x.route.id]:void 0,C=x.route.errorElement||cw,u&&(p<0&&y===0?(ug("route-fallback",!1,"No `HydrateFallback` element provided to render during initial hydration"),E=!0,S=null):p===y&&(E=!0,S=x.route.hydrateFallbackElement||null)));let P=t.concat(s.slice(0,y+1)),b=()=>{let A;return w?A=C:E?A=S:x.route.Component?A=D.createElement(x.route.Component,null):x.route.element?A=x.route.element:A=h,D.createElement(dw,{match:x,routeContext:{outlet:h,matches:P,isDataRoute:n!=null},children:A})};return n&&(x.route.ErrorBoundary||x.route.errorElement||y===0)?D.createElement(cg,{location:n.location,revalidation:n.revalidation,component:C,error:w,children:b(),routeContext:{outlet:null,matches:P,isDataRoute:!0},onError:f}):b()},null)}function Ad(r){return`${r} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`}function pw(r){let t=D.useContext(So);return qe(t,Ad(r)),t}function hw(r){let t=D.useContext(ui);return qe(t,Ad(r)),t}function mw(r){let t=D.useContext(pn);return qe(t,Ad(r)),t}function Pd(r){let t=mw(r),n=t.matches[t.matches.length-1];return qe(n.route.id,`${r} can only be used on routes that contain a unique "id"`),n.route.id}function gw(){return Pd("useRouteId")}function xw(){var a;let r=D.useContext(Rd),t=hw("useRouteError"),n=Pd("useRouteError");return r!==void 0?r:(a=t.errors)==null?void 0:a[n]}function vw(){let{router:r}=pw("useNavigate"),t=Pd("useNavigate"),n=D.useRef(!1);return sg(()=>{n.current=!0}),D.useCallback(async(i,s={})=>{bt(n.current,ig),n.current&&(typeof i=="number"?await r.navigate(i):await r.navigate(i,{fromRouteId:t,...s}))},[r,t])}var lh={};function ug(r,t,n){!t&&!lh[r]&&(lh[r]=!0,bt(!1,n))}var ch={};function uh(r,t){!r&&!ch[t]&&(ch[t]=!0,console.warn(t))}var yw="useOptimistic",dh=Kv[yw],ww=()=>{};function Ew(r){return dh?dh(r):[r,ww]}function Cw(r){let t={hasErrorBoundary:r.hasErrorBoundary||r.ErrorBoundary!=null||r.errorElement!=null};return r.Component&&(r.element&&bt(!1,"You should not include both `Component` and `element` on your route - `Component` will be used."),Object.assign(t,{element:D.createElement(r.Component),Component:void 0})),r.HydrateFallback&&(r.hydrateFallbackElement&&bt(!1,"You should not include both `HydrateFallback` and `hydrateFallbackElement` on your route - `HydrateFallback` will be used."),Object.assign(t,{hydrateFallbackElement:D.createElement(r.HydrateFallback),HydrateFallback:void 0})),r.ErrorBoundary&&(r.errorElement&&bt(!1,"You should not include both `ErrorBoundary` and `errorElement` on your route - `ErrorBoundary` will be used."),Object.assign(t,{errorElement:D.createElement(r.ErrorBoundary),ErrorBoundary:void 0})),t}var bw=["HydrateFallback","hydrateFallbackElement"],Sw=class{constructor(){this.status="pending",this.promise=new Promise((r,t)=>{this.resolve=n=>{this.status==="pending"&&(this.status="resolved",r(n))},this.reject=n=>{this.status==="pending"&&(this.status="rejected",t(n))}})}};function Rw({router:r,flushSync:t,onError:n,unstable_useTransitions:a}){a=Z1()||a;let[s,c]=D.useState(r.state),[u,p]=Ew(s),[f,h]=D.useState(),[x,y]=D.useState({isTransitioning:!1}),[w,E]=D.useState(),[C,S]=D.useState(),[P,b]=D.useState(),A=D.useRef(new Map),T=D.useCallback((N,{deletedFetchers:I,newErrors:L,flushSync:M,viewTransitionOpts:H})=>{L&&n&&Object.values(L).forEach(Z=>{var oe;return n(Z,{location:N.location,params:((oe=N.matches[0])==null?void 0:oe.params)??{},unstable_pattern:ci(N.matches)})}),N.fetchers.forEach((Z,oe)=>{Z.data!==void 0&&A.current.set(oe,Z.data)}),I.forEach(Z=>A.current.delete(Z)),uh(M===!1||t!=null,'You provided the `flushSync` option to a router update, but you are not using the `<RouterProvider>` from `react-router/dom` so `ReactDOM.flushSync()` is unavailable.  Please update your app to `import { RouterProvider } from "react-router/dom"` and ensure you have `react-dom` installed as a dependency to use the `flushSync` option.');let z=r.window!=null&&r.window.document!=null&&typeof r.window.document.startViewTransition=="function";if(uh(H==null||z,"You provided the `viewTransition` option to a router update, but you do not appear to be running in a DOM environment as `window.startViewTransition` is not available."),!H||!z){t&&M?t(()=>c(N)):a===!1?c(N):D.startTransition(()=>{a===!0&&p(Z=>fh(Z,N)),c(N)});return}if(t&&M){t(()=>{C&&(w==null||w.resolve(),C.skipTransition()),y({isTransitioning:!0,flushSync:!0,currentLocation:H.currentLocation,nextLocation:H.nextLocation})});let Z=r.window.document.startViewTransition(()=>{t(()=>c(N))});Z.finished.finally(()=>{t(()=>{E(void 0),S(void 0),h(void 0),y({isTransitioning:!1})})}),t(()=>S(Z));return}C?(w==null||w.resolve(),C.skipTransition(),b({state:N,currentLocation:H.currentLocation,nextLocation:H.nextLocation})):(h(N),y({isTransitioning:!0,flushSync:!1,currentLocation:H.currentLocation,nextLocation:H.nextLocation}))},[r.window,t,C,w,a,p,n]);D.useLayoutEffect(()=>r.subscribe(T),[r,T]),D.useEffect(()=>{x.isTransitioning&&!x.flushSync&&E(new Sw)},[x]),D.useEffect(()=>{if(w&&f&&r.window){let N=f,I=w.promise,L=r.window.document.startViewTransition(async()=>{a===!1?c(N):D.startTransition(()=>{a===!0&&p(M=>fh(M,N)),c(N)}),await I});L.finished.finally(()=>{E(void 0),S(void 0),h(void 0),y({isTransitioning:!1})}),S(L)}},[f,w,r.window,a,p]),D.useEffect(()=>{w&&f&&u.location.key===f.location.key&&w.resolve()},[w,C,u.location,f]),D.useEffect(()=>{!x.isTransitioning&&P&&(h(P.state),y({isTransitioning:!0,flushSync:!1,currentLocation:P.currentLocation,nextLocation:P.nextLocation}),b(void 0))},[x.isTransitioning,P]);let B=D.useMemo(()=>({createHref:r.createHref,encodeLocation:r.encodeLocation,go:N=>r.navigate(N),push:(N,I,L)=>r.navigate(N,{state:I,preventScrollReset:L==null?void 0:L.preventScrollReset}),replace:(N,I,L)=>r.navigate(N,{replace:!0,state:I,preventScrollReset:L==null?void 0:L.preventScrollReset})}),[r]),F=r.basename||"/",k=D.useMemo(()=>({router:r,navigator:B,static:!1,basename:F,onError:n}),[r,B,F,n]);return D.createElement(D.Fragment,null,D.createElement(So.Provider,{value:k},D.createElement(ui.Provider,{value:u},D.createElement(og.Provider,{value:A.current},D.createElement(Sd.Provider,{value:x},D.createElement(kw,{basename:F,location:u.location,navigationType:u.historyAction,navigator:B,unstable_useTransitions:a},D.createElement(Aw,{routes:r.routes,future:r.future,state:u,onError:n})))))),null)}function fh(r,t){return{...r,navigation:t.navigation.state!=="idle"?t.navigation:r.navigation,revalidation:t.revalidation!=="idle"?t.revalidation:r.revalidation,actionData:t.navigation.state!=="submitting"?t.actionData:r.actionData,fetchers:t.fetchers}}var Aw=D.memo(Pw);function Pw({routes:r,future:t,state:n,onError:a}){return sw(r,void 0,n,a,t)}function kw({basename:r="/",children:t=null,location:n,navigationType:a="POP",navigator:i,static:s=!1,unstable_useTransitions:c}){qe(!di(),"You cannot render a <Router> inside another <Router>. You should never have more than one in your app.");let u=r.replace(/^\/*/,"/"),p=D.useMemo(()=>({basename:u,navigator:i,static:s,unstable_useTransitions:c,future:{}}),[u,i,s,c]);typeof n=="string"&&(n=fn(n));let{pathname:f="/",search:h="",hash:x="",state:y=null,key:w="default"}=n,E=D.useMemo(()=>{let C=Rr(f,u);return C==null?null:{location:{pathname:C,search:h,hash:x,state:y,key:w},navigationType:a}},[u,f,h,x,y,w,a]);return bt(E!=null,`<Router basename="${u}"> is not able to match the URL "${f}${h}${x}" because it does not start with the basename, so the <Router> won't render anything.`),E==null?null:D.createElement(Ar.Provider,{value:p},D.createElement(Il.Provider,{children:t,value:E}))}var _s="get",Ls="application/x-www-form-urlencoded";function Bl(r){return typeof HTMLElement<"u"&&r instanceof HTMLElement}function Dw(r){return Bl(r)&&r.tagName.toLowerCase()==="button"}function Tw(r){return Bl(r)&&r.tagName.toLowerCase()==="form"}function _w(r){return Bl(r)&&r.tagName.toLowerCase()==="input"}function Lw(r){return!!(r.metaKey||r.altKey||r.ctrlKey||r.shiftKey)}function Fw(r,t){return r.button===0&&(!t||t==="_self")&&!Lw(r)}function fd(r=""){return new URLSearchParams(typeof r=="string"||Array.isArray(r)||r instanceof URLSearchParams?r:Object.keys(r).reduce((t,n)=>{let a=r[n];return t.concat(Array.isArray(a)?a.map(i=>[n,i]):[[n,a]])},[]))}function Nw(r,t){let n=fd(r);return t&&t.forEach((a,i)=>{n.has(i)||t.getAll(i).forEach(s=>{n.append(i,s)})}),n}var ys=null;function Iw(){if(ys===null)try{new FormData(document.createElement("form"),0),ys=!1}catch{ys=!0}return ys}var Bw=new Set(["application/x-www-form-urlencoded","multipart/form-data","text/plain"]);function Hu(r){return r!=null&&!Bw.has(r)?(bt(!1,`"${r}" is not a valid \`encType\` for \`<Form>\`/\`<fetcher.Form>\` and will default to "${Ls}"`),null):r}function jw(r,t){let n,a,i,s,c;if(Tw(r)){let u=r.getAttribute("action");a=u?Rr(u,t):null,n=r.getAttribute("method")||_s,i=Hu(r.getAttribute("enctype"))||Ls,s=new FormData(r)}else if(Dw(r)||_w(r)&&(r.type==="submit"||r.type==="image")){let u=r.form;if(u==null)throw new Error('Cannot submit a <button> or <input type="submit"> without a <form>');let p=r.getAttribute("formaction")||u.getAttribute("action");if(a=p?Rr(p,t):null,n=r.getAttribute("formmethod")||u.getAttribute("method")||_s,i=Hu(r.getAttribute("formenctype"))||Hu(u.getAttribute("enctype"))||Ls,s=new FormData(u,r),!Iw()){let{name:f,type:h,value:x}=r;if(h==="image"){let y=f?`${f}.`:"";s.append(`${y}x`,"0"),s.append(`${y}y`,"0")}else f&&s.append(f,x)}}else{if(Bl(r))throw new Error('Cannot submit element that is not <form>, <button>, or <input type="submit|image">');n=_s,a=null,i=Ls,c=r}return s&&i==="text/plain"&&(c=s,s=void 0),{action:a,method:n.toLowerCase(),encType:i,formData:s,body:c}}Object.getOwnPropertyNames(Object.prototype).sort().join("\0");function kd(r,t){if(r===!1||r===null||typeof r>"u")throw new Error(t)}function Ow(r,t,n,a){let i=typeof r=="string"?new URL(r,typeof window>"u"?"server://singlefetch/":window.location.origin):r;return n?i.pathname.endsWith("/")?i.pathname=`${i.pathname}_.${a}`:i.pathname=`${i.pathname}.${a}`:i.pathname==="/"?i.pathname=`_root.${a}`:t&&Rr(i.pathname,t)==="/"?i.pathname=`${t.replace(/\/$/,"")}/_root.${a}`:i.pathname=`${i.pathname.replace(/\/$/,"")}.${a}`,i}async function Mw(r,t){if(r.id in t)return t[r.id];try{let n=await import(r.module);return t[r.id]=n,n}catch(n){return console.error(`Error loading route module \`${r.module}\`, reloading page...`),console.error(n),window.__reactRouterContext&&window.__reactRouterContext.isSpaMode,window.location.reload(),new Promise(()=>{})}}function $w(r){return r==null?!1:r.href==null?r.rel==="preload"&&typeof r.imageSrcSet=="string"&&typeof r.imageSizes=="string":typeof r.rel=="string"&&typeof r.href=="string"}async function Uw(r,t,n){let a=await Promise.all(r.map(async i=>{let s=t.routes[i.route.id];if(s){let c=await Mw(s,n);return c.links?c.links():[]}return[]}));return Ww(a.flat(1).filter($w).filter(i=>i.rel==="stylesheet"||i.rel==="preload").map(i=>i.rel==="stylesheet"?{...i,rel:"prefetch",as:"style"}:{...i,rel:"prefetch"}))}function ph(r,t,n,a,i,s){let c=(p,f)=>n[f]?p.route.id!==n[f].route.id:!0,u=(p,f)=>{var h;return n[f].pathname!==p.pathname||((h=n[f].route.path)==null?void 0:h.endsWith("*"))&&n[f].params["*"]!==p.params["*"]};return s==="assets"?t.filter((p,f)=>c(p,f)||u(p,f)):s==="data"?t.filter((p,f)=>{var x;let h=a.routes[p.route.id];if(!h||!h.hasLoader)return!1;if(c(p,f)||u(p,f))return!0;if(p.route.shouldRevalidate){let y=p.route.shouldRevalidate({currentUrl:new URL(i.pathname+i.search+i.hash,window.origin),currentParams:((x=n[0])==null?void 0:x.params)||{},nextUrl:new URL(r,window.origin),nextParams:p.params,defaultShouldRevalidate:!0});if(typeof y=="boolean")return y}return!0}):[]}function zw(r,t,{includeHydrateFallback:n}={}){return Hw(r.map(a=>{let i=t.routes[a.route.id];if(!i)return[];let s=[i.module];return i.clientActionModule&&(s=s.concat(i.clientActionModule)),i.clientLoaderModule&&(s=s.concat(i.clientLoaderModule)),n&&i.hydrateFallbackModule&&(s=s.concat(i.hydrateFallbackModule)),i.imports&&(s=s.concat(i.imports)),s}).flat(1))}function Hw(r){return[...new Set(r)]}function Vw(r){let t={},n=Object.keys(r).sort();for(let a of n)t[a]=r[a];return t}function Ww(r,t){let n=new Set;return new Set(t),r.reduce((a,i)=>{let s=JSON.stringify(Vw(i));return n.has(s)||(n.add(s),a.push({key:s,link:i})),a},[])}function dg(){let r=D.useContext(So);return kd(r,"You must render this element inside a <DataRouterContext.Provider> element"),r}function Gw(){let r=D.useContext(ui);return kd(r,"You must render this element inside a <DataRouterStateContext.Provider> element"),r}var Dd=D.createContext(void 0);Dd.displayName="FrameworkContext";function fg(){let r=D.useContext(Dd);return kd(r,"You must render this element inside a <HydratedRouter> element"),r}function qw(r,t){let n=D.useContext(Dd),[a,i]=D.useState(!1),[s,c]=D.useState(!1),{onFocus:u,onBlur:p,onMouseEnter:f,onMouseLeave:h,onTouchStart:x}=t,y=D.useRef(null);D.useEffect(()=>{if(r==="render"&&c(!0),r==="viewport"){let C=P=>{P.forEach(b=>{c(b.isIntersecting)})},S=new IntersectionObserver(C,{threshold:.5});return y.current&&S.observe(y.current),()=>{S.disconnect()}}},[r]),D.useEffect(()=>{if(a){let C=setTimeout(()=>{c(!0)},100);return()=>{clearTimeout(C)}}},[a]);let w=()=>{i(!0)},E=()=>{i(!1),c(!1)};return n?r!=="intent"?[s,y,{}]:[s,y,{onFocus:Ha(u,w),onBlur:Ha(p,E),onMouseEnter:Ha(f,w),onMouseLeave:Ha(h,E),onTouchStart:Ha(x,w)}]:[!1,y,{}]}function Ha(r,t){return n=>{r&&r(n),n.defaultPrevented||t(n)}}function Kw({page:r,...t}){let{router:n}=dg(),a=D.useMemo(()=>Gn(n.routes,r,n.basename),[n.routes,r,n.basename]);return a?D.createElement(Yw,{page:r,matches:a,...t}):null}function Xw(r){let{manifest:t,routeModules:n}=fg(),[a,i]=D.useState([]);return D.useEffect(()=>{let s=!1;return Uw(r,t,n).then(c=>{s||i(c)}),()=>{s=!0}},[r,t,n]),a}function Yw({page:r,matches:t,...n}){let a=hn(),{future:i,manifest:s,routeModules:c}=fg(),{basename:u}=dg(),{loaderData:p,matches:f}=Gw(),h=D.useMemo(()=>ph(r,t,f,s,a,"data"),[r,t,f,s,a]),x=D.useMemo(()=>ph(r,t,f,s,a,"assets"),[r,t,f,s,a]),y=D.useMemo(()=>{if(r===a.pathname+a.search+a.hash)return[];let C=new Set,S=!1;if(t.forEach(b=>{var T;let A=s.routes[b.route.id];!A||!A.hasLoader||(!h.some(B=>B.route.id===b.route.id)&&b.route.id in p&&((T=c[b.route.id])!=null&&T.shouldRevalidate)||A.hasClientLoader?S=!0:C.add(b.route.id))}),C.size===0)return[];let P=Ow(r,u,i.unstable_trailingSlashAwareDataRequests,"data");return S&&C.size>0&&P.searchParams.set("_routes",t.filter(b=>C.has(b.route.id)).map(b=>b.route.id).join(",")),[P.pathname+P.search]},[u,i.unstable_trailingSlashAwareDataRequests,p,a,s,h,t,r,c]),w=D.useMemo(()=>zw(x,s),[x,s]),E=Xw(x);return D.createElement(D.Fragment,null,y.map(C=>D.createElement("link",{key:C,rel:"prefetch",as:"fetch",href:C,...n})),w.map(C=>D.createElement("link",{key:C,rel:"modulepreload",href:C,...n})),E.map(({key:C,link:S})=>D.createElement("link",{key:C,nonce:n.nonce,...S})))}function Qw(...r){return t=>{r.forEach(n=>{typeof n=="function"?n(t):n!=null&&(n.current=t)})}}var Jw=typeof window<"u"&&typeof window.document<"u"&&typeof window.document.createElement<"u";try{Jw&&(window.__reactRouterVersion="7.12.0")}catch{}function Zw(r,t){return k1({basename:t==null?void 0:t.basename,getContext:t==null?void 0:t.getContext,future:t==null?void 0:t.future,history:Vy({window:t==null?void 0:t.window}),hydrationData:e2(),routes:r,mapRouteProperties:Cw,hydrationRouteProperties:bw,dataStrategy:t==null?void 0:t.dataStrategy,patchRoutesOnNavigation:t==null?void 0:t.patchRoutesOnNavigation,window:t==null?void 0:t.window,unstable_instrumentations:t==null?void 0:t.unstable_instrumentations}).initialize()}function e2(){let r=window==null?void 0:window.__staticRouterHydrationData;return r&&r.errors&&(r={...r,errors:t2(r.errors)}),r}function t2(r){if(!r)return null;let t=Object.entries(r),n={};for(let[a,i]of t)if(i&&i.__type==="RouteErrorResponse")n[a]=new li(i.status,i.statusText,i.data,i.internal===!0);else if(i&&i.__type==="Error"){if(i.__subType){let s=window[i.__subType];if(typeof s=="function")try{let c=new s(i.message);c.stack="",n[a]=c}catch{}}if(n[a]==null){let s=new Error(i.message);s.stack="",n[a]=s}}else n[a]=i;return n}var pg=/^(?:[a-z][a-z0-9+.-]*:|\/\/)/i,Xn=D.forwardRef(function({onClick:t,discover:n="render",prefetch:a="none",relative:i,reloadDocument:s,replace:c,state:u,target:p,to:f,preventScrollReset:h,viewTransition:x,unstable_defaultShouldRevalidate:y,...w},E){let{basename:C,unstable_useTransitions:S}=D.useContext(Ar),P=typeof f=="string"&&pg.test(f),b=Wm(f,C);f=b.to;let A=aw(f,{relative:i}),[T,B,F]=qw(a,w),k=a2(f,{replace:c,state:u,target:p,preventScrollReset:h,relative:i,viewTransition:x,unstable_defaultShouldRevalidate:y,unstable_useTransitions:S});function N(L){t&&t(L),L.defaultPrevented||k(L)}let I=D.createElement("a",{...w,...F,href:b.absoluteURL||A,onClick:b.isExternal||s?t:N,ref:Qw(E,B),target:p,"data-discover":!P&&n==="render"?"true":void 0});return T&&!P?D.createElement(D.Fragment,null,I,D.createElement(Kw,{page:A})):I});Xn.displayName="Link";var r2=D.forwardRef(function({"aria-current":t="page",caseSensitive:n=!1,className:a="",end:i=!1,style:s,to:c,viewTransition:u,children:p,...f},h){let x=fi(c,{relative:f.relative}),y=hn(),w=D.useContext(ui),{navigator:E,basename:C}=D.useContext(Ar),S=w!=null&&d2(x)&&u===!0,P=E.encodeLocation?E.encodeLocation(x).pathname:x.pathname,b=y.pathname,A=w&&w.navigation&&w.navigation.location?w.navigation.location.pathname:null;n||(b=b.toLowerCase(),A=A?A.toLowerCase():null,P=P.toLowerCase()),A&&C&&(A=Rr(A,C)||A);const T=P!=="/"&&P.endsWith("/")?P.length-1:P.length;let B=b===P||!i&&b.startsWith(P)&&b.charAt(T)==="/",F=A!=null&&(A===P||!i&&A.startsWith(P)&&A.charAt(P.length)==="/"),k={isActive:B,isPending:F,isTransitioning:S},N=B?t:void 0,I;typeof a=="function"?I=a(k):I=[a,B?"active":null,F?"pending":null,S?"transitioning":null].filter(Boolean).join(" ");let L=typeof s=="function"?s(k):s;return D.createElement(Xn,{...f,"aria-current":N,className:I,ref:h,style:L,to:c,viewTransition:u},typeof p=="function"?p(k):p)});r2.displayName="NavLink";var n2=D.forwardRef(({discover:r="render",fetcherKey:t,navigate:n,reloadDocument:a,replace:i,state:s,method:c=_s,action:u,onSubmit:p,relative:f,preventScrollReset:h,viewTransition:x,unstable_defaultShouldRevalidate:y,...w},E)=>{let{unstable_useTransitions:C}=D.useContext(Ar),S=c2(),P=u2(u,{relative:f}),b=c.toLowerCase()==="get"?"get":"post",A=typeof u=="string"&&pg.test(u),T=B=>{if(p&&p(B),B.defaultPrevented)return;B.preventDefault();let F=B.nativeEvent.submitter,k=(F==null?void 0:F.getAttribute("formmethod"))||c,N=()=>S(F||B.currentTarget,{fetcherKey:t,method:k,navigate:n,replace:i,state:s,relative:f,preventScrollReset:h,viewTransition:x,unstable_defaultShouldRevalidate:y});C&&n!==!1?D.startTransition(()=>N()):N()};return D.createElement("form",{ref:E,method:b,action:P,onSubmit:a?p:T,...w,"data-discover":!A&&r==="render"?"true":void 0})});n2.displayName="Form";function o2(r){return`${r} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`}function hg(r){let t=D.useContext(So);return qe(t,o2(r)),t}function a2(r,{target:t,replace:n,state:a,preventScrollReset:i,relative:s,viewTransition:c,unstable_defaultShouldRevalidate:u,unstable_useTransitions:p}={}){let f=lg(),h=hn(),x=fi(r,{relative:s});return D.useCallback(y=>{if(Fw(y,t)){y.preventDefault();let w=n!==void 0?n:qr(h)===qr(x),E=()=>f(r,{replace:w,state:a,preventScrollReset:i,relative:s,viewTransition:c,unstable_defaultShouldRevalidate:u});p?D.startTransition(()=>E()):E()}},[h,f,x,n,a,t,r,i,s,c,u,p])}function i2(r){bt(typeof URLSearchParams<"u","You cannot use the `useSearchParams` hook in a browser that does not support the URLSearchParams API. If you need to support Internet Explorer 11, we recommend you load a polyfill such as https://github.com/ungap/url-search-params.");let t=D.useRef(fd(r)),n=D.useRef(!1),a=hn(),i=D.useMemo(()=>Nw(a.search,n.current?null:t.current),[a.search]),s=lg(),c=D.useCallback((u,p)=>{const f=fd(typeof u=="function"?u(new URLSearchParams(i)):u);n.current=!0,s("?"+f,p)},[s,i]);return[i,c]}var s2=0,l2=()=>`__${String(++s2)}__`;function c2(){let{router:r}=hg("useSubmit"),{basename:t}=D.useContext(Ar),n=gw(),a=r.fetch,i=r.navigate;return D.useCallback(async(s,c={})=>{let{action:u,method:p,encType:f,formData:h,body:x}=jw(s,t);if(c.navigate===!1){let y=c.fetcherKey||l2();await a(y,n,c.action||u,{unstable_defaultShouldRevalidate:c.unstable_defaultShouldRevalidate,preventScrollReset:c.preventScrollReset,formData:h,body:x,formMethod:c.method||p,formEncType:c.encType||f,flushSync:c.flushSync})}else await i(c.action||u,{unstable_defaultShouldRevalidate:c.unstable_defaultShouldRevalidate,preventScrollReset:c.preventScrollReset,formData:h,body:x,formMethod:c.method||p,formEncType:c.encType||f,replace:c.replace,state:c.state,fromRouteId:n,flushSync:c.flushSync,viewTransition:c.viewTransition})},[a,i,t,n])}function u2(r,{relative:t}={}){let{basename:n}=D.useContext(Ar),a=D.useContext(pn);qe(a,"useFormAction must be used inside a RouteContext");let[i]=a.matches.slice(-1),s={...fi(r||".",{relative:t})},c=hn();if(r==null){s.search=c.search;let u=new URLSearchParams(s.search),p=u.getAll("index");if(p.some(h=>h==="")){u.delete("index"),p.filter(x=>x).forEach(x=>u.append("index",x));let h=u.toString();s.search=h?`?${h}`:""}}return(!r||r===".")&&i.route.index&&(s.search=s.search?s.search.replace(/^\?/,"?index&"):"?index"),n!=="/"&&(s.pathname=s.pathname==="/"?n:Gr([n,s.pathname])),qr(s)}function d2(r,{relative:t}={}){let n=D.useContext(Sd);qe(n!=null,"`useViewTransitionState` must be used within `react-router-dom`'s `RouterProvider`.  Did you accidentally import `RouterProvider` from `react-router`?");let{basename:a}=hg("useViewTransitionState"),i=fi(r,{relative:t});if(!n.isTransitioning)return!1;let s=Rr(n.currentLocation.pathname,a)||n.currentLocation.pathname,c=Rr(n.nextLocation.pathname,a)||n.nextLocation.pathname;return Sl(i.pathname,c)!=null||Sl(i.pathname,s)!=null}var f2=_m();function p2(r){return D.createElement(Rw,{flushSync:f2.flushSync,...r})}function jl({children:r,className:t=""}){const[n,a]=D.useState(!1),i=D.useRef(null);return D.useEffect(()=>{window.matchMedia("(prefers-reduced-motion: reduce)").matches?a(!0):requestAnimationFrame(()=>{a(!0)})},[]),g.jsx("div",{ref:i,className:`page-enter ${n?"opacity-100":"opacity-0"} ${t}`,children:r})}const hh=[{path:"/",labelKey:"nav.home",showInNav:!0},{path:"/neon-lab",labelKey:"nav.neonLab",showInNav:!0},{path:"/demos",labelKey:"nav.demos",showInNav:!0}],h2="modulepreload",m2=function(r){return"/neon/"+r},mh={},gh=function(t,n,a){let i=Promise.resolve();if(n&&n.length>0){let c=function(f){return Promise.all(f.map(h=>Promise.resolve(h).then(x=>({status:"fulfilled",value:x}),x=>({status:"rejected",reason:x}))))};document.getElementsByTagName("link");const u=document.querySelector("meta[property=csp-nonce]"),p=(u==null?void 0:u.nonce)||(u==null?void 0:u.getAttribute("nonce"));i=c(n.map(f=>{if(f=m2(f),f in mh)return;mh[f]=!0;const h=f.endsWith(".css"),x=h?'[rel="stylesheet"]':"";if(document.querySelector(`link[href="${f}"]${x}`))return;const y=document.createElement("link");if(y.rel=h?"stylesheet":h2,h||(y.as="script"),y.crossOrigin="",y.href=f,p&&y.setAttribute("nonce",p),document.head.appendChild(y),h)return new Promise((w,E)=>{y.addEventListener("load",w),y.addEventListener("error",()=>E(new Error(`Unable to preload CSS for ${f}`)))})}))}function s(c){const u=new Event("vite:preloadError",{cancelable:!0});if(u.payload=c,window.dispatchEvent(u),!u.defaultPrevented)throw c}return i.then(c=>{for(const u of c||[])u.status==="rejected"&&s(u.reason);return t().catch(s)})},xh=r=>{let t;const n=new Set,a=(f,h)=>{const x=typeof f=="function"?f(t):f;if(!Object.is(x,t)){const y=t;t=h??(typeof x!="object"||x===null)?x:Object.assign({},t,x),n.forEach(w=>w(t,y))}},i=()=>t,u={setState:a,getState:i,getInitialState:()=>p,subscribe:f=>(n.add(f),()=>n.delete(f))},p=t=r(a,i,u);return u},g2=(r=>r?xh(r):xh),x2=r=>r;function v2(r,t=x2){const n=Wa.useSyncExternalStore(r.subscribe,Wa.useCallback(()=>t(r.getState()),[r,t]),Wa.useCallback(()=>t(r.getInitialState()),[r,t]));return Wa.useDebugValue(n),n}const vh=r=>{const t=g2(r),n=a=>v2(t,a);return Object.assign(n,t),n},mg=(r=>r?vh(r):vh),Ir={MAX_IMAGE_SIZE:10*1024*1024,MAX_ATTACHMENTS:5,IMAGE_SHORT_EDGE_LIMIT:1080,ACCEPTED_IMAGE_FORMATS:["image/png","image/jpeg","image/webp"],ACCEPTED_DOCUMENT_FORMATS:["text/plain","text/markdown"],ACCEPTED_DOCUMENT_EXTENSIONS:[".txt",".md"]},Vu={INVALID_IMAGE_FORMAT:"error.attachment.invalidImageFormat",INVALID_DOCUMENT_FORMAT:"error.attachment.invalidDocumentFormat",FILE_TOO_LARGE:"error.attachment.fileTooLarge",EMPTY_DOCUMENT:"error.attachment.emptyDocument",READ_ERROR:"error.attachment.readError",MAX_ATTACHMENTS_REACHED:"error.attachment.maxAttachmentsReached",CORRUPTED_FILE:"error.attachment.corruptedFile"},yh=30,gg="history.newConversation",y2="https://api.openai.com/v1",Fs={MAX_FIX_ATTEMPTS:3},wh={SyntaxError:"error.friendly.syntaxError",ReferenceError:"error.friendly.referenceError",TypeError:"error.friendly.typeError",RangeError:"error.friendly.rangeError",EvalError:"error.friendly.evalError",URIError:"error.friendly.uriError",default:"error.friendly.default"};function xg(r){return wh[r]||wh.default}function vg(){return`error_${Date.now()}_${Math.random().toString(36).substring(2,9)}`}const na={MAX_FILE_SIZE:10*1024*1024,MAX_DIMENSION:4096,ACCEPTED_FORMATS:["image/png","image/jpeg"],ACCEPTED_EXTENSIONS:[".png",".jpg",".jpeg"]},Rl={PNG:[137,80,78,71,13,10,26,10],JPEG:[255,216,255]},Td="__PLACEHOLDER__",w2={INVALID_FORMAT:"error.image.invalidFormat",FILE_TOO_LARGE:"error.image.fileTooLarge",DIMENSION_TOO_LARGE:"error.image.dimensionTooLarge",CORRUPTED_FILE:"error.image.corruptedFile",READ_ERROR:"error.image.readError"},Ol={MAX_FILE_SIZE:50*1024*1024,MAX_DURATION:60*1e3,ACCEPTED_FORMATS:["video/mp4","video/webm"],ACCEPTED_EXTENSIONS:[".mp4",".webm"]},E2={INVALID_FORMAT:"error.video.invalidFormat",FILE_TOO_LARGE:"error.video.fileTooLarge",DURATION_TOO_LONG:"error.video.durationTooLong",RESOLUTION_TOO_LARGE:"error.video.resolutionTooLarge",CORRUPTED_FILE:"error.video.corruptedFile",LOAD_ERROR:"error.video.loadError"},_d="__VIDEO_PLACEHOLDER__",Ns={status:"idle",config:null,progress:0,error:null},Wu=2e3,C2=5e3;var Is={exports:{}};function b2(r){throw new Error('Could not dynamically require "'+r+'". Please configure the dynamicRequireTargets or/and ignoreDynamicRequires option of @rollup/plugin-commonjs appropriately for this require call to work.')}var Bs={exports:{}};const S2={},R2=Object.freeze(Object.defineProperty({__proto__:null,default:S2},Symbol.toStringTag,{value:"Module"})),A2=Vv(R2);var P2=Bs.exports,Eh;function Je(){return Eh||(Eh=1,(function(r,t){(function(n,a){r.exports=a()})(P2,function(){var n=n||(function(a,i){var s;if(typeof window<"u"&&window.crypto&&(s=window.crypto),typeof self<"u"&&self.crypto&&(s=self.crypto),typeof globalThis<"u"&&globalThis.crypto&&(s=globalThis.crypto),!s&&typeof window<"u"&&window.msCrypto&&(s=window.msCrypto),!s&&typeof Pu<"u"&&Pu.crypto&&(s=Pu.crypto),!s&&typeof b2=="function")try{s=A2}catch{}var c=function(){if(s){if(typeof s.getRandomValues=="function")try{return s.getRandomValues(new Uint32Array(1))[0]}catch{}if(typeof s.randomBytes=="function")try{return s.randomBytes(4).readInt32LE()}catch{}}throw new Error("Native crypto module could not be used to get secure random number.")},u=Object.create||(function(){function b(){}return function(A){var T;return b.prototype=A,T=new b,b.prototype=null,T}})(),p={},f=p.lib={},h=f.Base=(function(){return{extend:function(b){var A=u(this);return b&&A.mixIn(b),(!A.hasOwnProperty("init")||this.init===A.init)&&(A.init=function(){A.$super.init.apply(this,arguments)}),A.init.prototype=A,A.$super=this,A},create:function(){var b=this.extend();return b.init.apply(b,arguments),b},init:function(){},mixIn:function(b){for(var A in b)b.hasOwnProperty(A)&&(this[A]=b[A]);b.hasOwnProperty("toString")&&(this.toString=b.toString)},clone:function(){return this.init.prototype.extend(this)}}})(),x=f.WordArray=h.extend({init:function(b,A){b=this.words=b||[],A!=i?this.sigBytes=A:this.sigBytes=b.length*4},toString:function(b){return(b||w).stringify(this)},concat:function(b){var A=this.words,T=b.words,B=this.sigBytes,F=b.sigBytes;if(this.clamp(),B%4)for(var k=0;k<F;k++){var N=T[k>>>2]>>>24-k%4*8&255;A[B+k>>>2]|=N<<24-(B+k)%4*8}else for(var I=0;I<F;I+=4)A[B+I>>>2]=T[I>>>2];return this.sigBytes+=F,this},clamp:function(){var b=this.words,A=this.sigBytes;b[A>>>2]&=4294967295<<32-A%4*8,b.length=a.ceil(A/4)},clone:function(){var b=h.clone.call(this);return b.words=this.words.slice(0),b},random:function(b){for(var A=[],T=0;T<b;T+=4)A.push(c());return new x.init(A,b)}}),y=p.enc={},w=y.Hex={stringify:function(b){for(var A=b.words,T=b.sigBytes,B=[],F=0;F<T;F++){var k=A[F>>>2]>>>24-F%4*8&255;B.push((k>>>4).toString(16)),B.push((k&15).toString(16))}return B.join("")},parse:function(b){for(var A=b.length,T=[],B=0;B<A;B+=2)T[B>>>3]|=parseInt(b.substr(B,2),16)<<24-B%8*4;return new x.init(T,A/2)}},E=y.Latin1={stringify:function(b){for(var A=b.words,T=b.sigBytes,B=[],F=0;F<T;F++){var k=A[F>>>2]>>>24-F%4*8&255;B.push(String.fromCharCode(k))}return B.join("")},parse:function(b){for(var A=b.length,T=[],B=0;B<A;B++)T[B>>>2]|=(b.charCodeAt(B)&255)<<24-B%4*8;return new x.init(T,A)}},C=y.Utf8={stringify:function(b){try{return decodeURIComponent(escape(E.stringify(b)))}catch{throw new Error("Malformed UTF-8 data")}},parse:function(b){return E.parse(unescape(encodeURIComponent(b)))}},S=f.BufferedBlockAlgorithm=h.extend({reset:function(){this._data=new x.init,this._nDataBytes=0},_append:function(b){typeof b=="string"&&(b=C.parse(b)),this._data.concat(b),this._nDataBytes+=b.sigBytes},_process:function(b){var A,T=this._data,B=T.words,F=T.sigBytes,k=this.blockSize,N=k*4,I=F/N;b?I=a.ceil(I):I=a.max((I|0)-this._minBufferSize,0);var L=I*k,M=a.min(L*4,F);if(L){for(var H=0;H<L;H+=k)this._doProcessBlock(B,H);A=B.splice(0,L),T.sigBytes-=M}return new x.init(A,M)},clone:function(){var b=h.clone.call(this);return b._data=this._data.clone(),b},_minBufferSize:0});f.Hasher=S.extend({cfg:h.extend(),init:function(b){this.cfg=this.cfg.extend(b),this.reset()},reset:function(){S.reset.call(this),this._doReset()},update:function(b){return this._append(b),this._process(),this},finalize:function(b){b&&this._append(b);var A=this._doFinalize();return A},blockSize:16,_createHelper:function(b){return function(A,T){return new b.init(T).finalize(A)}},_createHmacHelper:function(b){return function(A,T){return new P.HMAC.init(b,T).finalize(A)}}});var P=p.algo={};return p})(Math);return n})})(Bs)),Bs.exports}var js={exports:{}},k2=js.exports,Ch;function Ml(){return Ch||(Ch=1,(function(r,t){(function(n,a){r.exports=a(Je())})(k2,function(n){return(function(a){var i=n,s=i.lib,c=s.Base,u=s.WordArray,p=i.x64={};p.Word=c.extend({init:function(f,h){this.high=f,this.low=h}}),p.WordArray=c.extend({init:function(f,h){f=this.words=f||[],h!=a?this.sigBytes=h:this.sigBytes=f.length*8},toX32:function(){for(var f=this.words,h=f.length,x=[],y=0;y<h;y++){var w=f[y];x.push(w.high),x.push(w.low)}return u.create(x,this.sigBytes)},clone:function(){for(var f=c.clone.call(this),h=f.words=this.words.slice(0),x=h.length,y=0;y<x;y++)h[y]=h[y].clone();return f}})})(),n})})(js)),js.exports}var Os={exports:{}},D2=Os.exports,bh;function T2(){return bh||(bh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(D2,function(n){return(function(){if(typeof ArrayBuffer=="function"){var a=n,i=a.lib,s=i.WordArray,c=s.init,u=s.init=function(p){if(p instanceof ArrayBuffer&&(p=new Uint8Array(p)),(p instanceof Int8Array||typeof Uint8ClampedArray<"u"&&p instanceof Uint8ClampedArray||p instanceof Int16Array||p instanceof Uint16Array||p instanceof Int32Array||p instanceof Uint32Array||p instanceof Float32Array||p instanceof Float64Array)&&(p=new Uint8Array(p.buffer,p.byteOffset,p.byteLength)),p instanceof Uint8Array){for(var f=p.byteLength,h=[],x=0;x<f;x++)h[x>>>2]|=p[x]<<24-x%4*8;c.call(this,h,f)}else c.apply(this,arguments)};u.prototype=s}})(),n.lib.WordArray})})(Os)),Os.exports}var Ms={exports:{}},_2=Ms.exports,Sh;function L2(){return Sh||(Sh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(_2,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=a.enc;c.Utf16=c.Utf16BE={stringify:function(p){for(var f=p.words,h=p.sigBytes,x=[],y=0;y<h;y+=2){var w=f[y>>>2]>>>16-y%4*8&65535;x.push(String.fromCharCode(w))}return x.join("")},parse:function(p){for(var f=p.length,h=[],x=0;x<f;x++)h[x>>>1]|=p.charCodeAt(x)<<16-x%2*16;return s.create(h,f*2)}},c.Utf16LE={stringify:function(p){for(var f=p.words,h=p.sigBytes,x=[],y=0;y<h;y+=2){var w=u(f[y>>>2]>>>16-y%4*8&65535);x.push(String.fromCharCode(w))}return x.join("")},parse:function(p){for(var f=p.length,h=[],x=0;x<f;x++)h[x>>>1]|=u(p.charCodeAt(x)<<16-x%2*16);return s.create(h,f*2)}};function u(p){return p<<8&4278255360|p>>>8&16711935}})(),n.enc.Utf16})})(Ms)),Ms.exports}var $s={exports:{}},F2=$s.exports,Rh;function Ro(){return Rh||(Rh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(F2,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=a.enc;c.Base64={stringify:function(p){var f=p.words,h=p.sigBytes,x=this._map;p.clamp();for(var y=[],w=0;w<h;w+=3)for(var E=f[w>>>2]>>>24-w%4*8&255,C=f[w+1>>>2]>>>24-(w+1)%4*8&255,S=f[w+2>>>2]>>>24-(w+2)%4*8&255,P=E<<16|C<<8|S,b=0;b<4&&w+b*.75<h;b++)y.push(x.charAt(P>>>6*(3-b)&63));var A=x.charAt(64);if(A)for(;y.length%4;)y.push(A);return y.join("")},parse:function(p){var f=p.length,h=this._map,x=this._reverseMap;if(!x){x=this._reverseMap=[];for(var y=0;y<h.length;y++)x[h.charCodeAt(y)]=y}var w=h.charAt(64);if(w){var E=p.indexOf(w);E!==-1&&(f=E)}return u(p,f,x)},_map:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="};function u(p,f,h){for(var x=[],y=0,w=0;w<f;w++)if(w%4){var E=h[p.charCodeAt(w-1)]<<w%4*2,C=h[p.charCodeAt(w)]>>>6-w%4*2,S=E|C;x[y>>>2]|=S<<24-y%4*8,y++}return s.create(x,y)}})(),n.enc.Base64})})($s)),$s.exports}var Us={exports:{}},N2=Us.exports,Ah;function I2(){return Ah||(Ah=1,(function(r,t){(function(n,a){r.exports=a(Je())})(N2,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=a.enc;c.Base64url={stringify:function(p,f){f===void 0&&(f=!0);var h=p.words,x=p.sigBytes,y=f?this._safe_map:this._map;p.clamp();for(var w=[],E=0;E<x;E+=3)for(var C=h[E>>>2]>>>24-E%4*8&255,S=h[E+1>>>2]>>>24-(E+1)%4*8&255,P=h[E+2>>>2]>>>24-(E+2)%4*8&255,b=C<<16|S<<8|P,A=0;A<4&&E+A*.75<x;A++)w.push(y.charAt(b>>>6*(3-A)&63));var T=y.charAt(64);if(T)for(;w.length%4;)w.push(T);return w.join("")},parse:function(p,f){f===void 0&&(f=!0);var h=p.length,x=f?this._safe_map:this._map,y=this._reverseMap;if(!y){y=this._reverseMap=[];for(var w=0;w<x.length;w++)y[x.charCodeAt(w)]=w}var E=x.charAt(64);if(E){var C=p.indexOf(E);C!==-1&&(h=C)}return u(p,h,y)},_map:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",_safe_map:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"};function u(p,f,h){for(var x=[],y=0,w=0;w<f;w++)if(w%4){var E=h[p.charCodeAt(w-1)]<<w%4*2,C=h[p.charCodeAt(w)]>>>6-w%4*2,S=E|C;x[y>>>2]|=S<<24-y%4*8,y++}return s.create(x,y)}})(),n.enc.Base64url})})(Us)),Us.exports}var zs={exports:{}},B2=zs.exports,Ph;function Ao(){return Ph||(Ph=1,(function(r,t){(function(n,a){r.exports=a(Je())})(B2,function(n){return(function(a){var i=n,s=i.lib,c=s.WordArray,u=s.Hasher,p=i.algo,f=[];(function(){for(var C=0;C<64;C++)f[C]=a.abs(a.sin(C+1))*4294967296|0})();var h=p.MD5=u.extend({_doReset:function(){this._hash=new c.init([1732584193,4023233417,2562383102,271733878])},_doProcessBlock:function(C,S){for(var P=0;P<16;P++){var b=S+P,A=C[b];C[b]=(A<<8|A>>>24)&16711935|(A<<24|A>>>8)&4278255360}var T=this._hash.words,B=C[S+0],F=C[S+1],k=C[S+2],N=C[S+3],I=C[S+4],L=C[S+5],M=C[S+6],H=C[S+7],z=C[S+8],Z=C[S+9],oe=C[S+10],re=C[S+11],se=C[S+12],Y=C[S+13],te=C[S+14],ee=C[S+15],_=T[0],$=T[1],G=T[2],K=T[3];_=x(_,$,G,K,B,7,f[0]),K=x(K,_,$,G,F,12,f[1]),G=x(G,K,_,$,k,17,f[2]),$=x($,G,K,_,N,22,f[3]),_=x(_,$,G,K,I,7,f[4]),K=x(K,_,$,G,L,12,f[5]),G=x(G,K,_,$,M,17,f[6]),$=x($,G,K,_,H,22,f[7]),_=x(_,$,G,K,z,7,f[8]),K=x(K,_,$,G,Z,12,f[9]),G=x(G,K,_,$,oe,17,f[10]),$=x($,G,K,_,re,22,f[11]),_=x(_,$,G,K,se,7,f[12]),K=x(K,_,$,G,Y,12,f[13]),G=x(G,K,_,$,te,17,f[14]),$=x($,G,K,_,ee,22,f[15]),_=y(_,$,G,K,F,5,f[16]),K=y(K,_,$,G,M,9,f[17]),G=y(G,K,_,$,re,14,f[18]),$=y($,G,K,_,B,20,f[19]),_=y(_,$,G,K,L,5,f[20]),K=y(K,_,$,G,oe,9,f[21]),G=y(G,K,_,$,ee,14,f[22]),$=y($,G,K,_,I,20,f[23]),_=y(_,$,G,K,Z,5,f[24]),K=y(K,_,$,G,te,9,f[25]),G=y(G,K,_,$,N,14,f[26]),$=y($,G,K,_,z,20,f[27]),_=y(_,$,G,K,Y,5,f[28]),K=y(K,_,$,G,k,9,f[29]),G=y(G,K,_,$,H,14,f[30]),$=y($,G,K,_,se,20,f[31]),_=w(_,$,G,K,L,4,f[32]),K=w(K,_,$,G,z,11,f[33]),G=w(G,K,_,$,re,16,f[34]),$=w($,G,K,_,te,23,f[35]),_=w(_,$,G,K,F,4,f[36]),K=w(K,_,$,G,I,11,f[37]),G=w(G,K,_,$,H,16,f[38]),$=w($,G,K,_,oe,23,f[39]),_=w(_,$,G,K,Y,4,f[40]),K=w(K,_,$,G,B,11,f[41]),G=w(G,K,_,$,N,16,f[42]),$=w($,G,K,_,M,23,f[43]),_=w(_,$,G,K,Z,4,f[44]),K=w(K,_,$,G,se,11,f[45]),G=w(G,K,_,$,ee,16,f[46]),$=w($,G,K,_,k,23,f[47]),_=E(_,$,G,K,B,6,f[48]),K=E(K,_,$,G,H,10,f[49]),G=E(G,K,_,$,te,15,f[50]),$=E($,G,K,_,L,21,f[51]),_=E(_,$,G,K,se,6,f[52]),K=E(K,_,$,G,N,10,f[53]),G=E(G,K,_,$,oe,15,f[54]),$=E($,G,K,_,F,21,f[55]),_=E(_,$,G,K,z,6,f[56]),K=E(K,_,$,G,ee,10,f[57]),G=E(G,K,_,$,M,15,f[58]),$=E($,G,K,_,Y,21,f[59]),_=E(_,$,G,K,I,6,f[60]),K=E(K,_,$,G,re,10,f[61]),G=E(G,K,_,$,k,15,f[62]),$=E($,G,K,_,Z,21,f[63]),T[0]=T[0]+_|0,T[1]=T[1]+$|0,T[2]=T[2]+G|0,T[3]=T[3]+K|0},_doFinalize:function(){var C=this._data,S=C.words,P=this._nDataBytes*8,b=C.sigBytes*8;S[b>>>5]|=128<<24-b%32;var A=a.floor(P/4294967296),T=P;S[(b+64>>>9<<4)+15]=(A<<8|A>>>24)&16711935|(A<<24|A>>>8)&4278255360,S[(b+64>>>9<<4)+14]=(T<<8|T>>>24)&16711935|(T<<24|T>>>8)&4278255360,C.sigBytes=(S.length+1)*4,this._process();for(var B=this._hash,F=B.words,k=0;k<4;k++){var N=F[k];F[k]=(N<<8|N>>>24)&16711935|(N<<24|N>>>8)&4278255360}return B},clone:function(){var C=u.clone.call(this);return C._hash=this._hash.clone(),C}});function x(C,S,P,b,A,T,B){var F=C+(S&P|~S&b)+A+B;return(F<<T|F>>>32-T)+S}function y(C,S,P,b,A,T,B){var F=C+(S&b|P&~b)+A+B;return(F<<T|F>>>32-T)+S}function w(C,S,P,b,A,T,B){var F=C+(S^P^b)+A+B;return(F<<T|F>>>32-T)+S}function E(C,S,P,b,A,T,B){var F=C+(P^(S|~b))+A+B;return(F<<T|F>>>32-T)+S}i.MD5=u._createHelper(h),i.HmacMD5=u._createHmacHelper(h)})(Math),n.MD5})})(zs)),zs.exports}var Hs={exports:{}},j2=Hs.exports,kh;function yg(){return kh||(kh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(j2,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=i.Hasher,u=a.algo,p=[],f=u.SHA1=c.extend({_doReset:function(){this._hash=new s.init([1732584193,4023233417,2562383102,271733878,3285377520])},_doProcessBlock:function(h,x){for(var y=this._hash.words,w=y[0],E=y[1],C=y[2],S=y[3],P=y[4],b=0;b<80;b++){if(b<16)p[b]=h[x+b]|0;else{var A=p[b-3]^p[b-8]^p[b-14]^p[b-16];p[b]=A<<1|A>>>31}var T=(w<<5|w>>>27)+P+p[b];b<20?T+=(E&C|~E&S)+1518500249:b<40?T+=(E^C^S)+1859775393:b<60?T+=(E&C|E&S|C&S)-1894007588:T+=(E^C^S)-899497514,P=S,S=C,C=E<<30|E>>>2,E=w,w=T}y[0]=y[0]+w|0,y[1]=y[1]+E|0,y[2]=y[2]+C|0,y[3]=y[3]+S|0,y[4]=y[4]+P|0},_doFinalize:function(){var h=this._data,x=h.words,y=this._nDataBytes*8,w=h.sigBytes*8;return x[w>>>5]|=128<<24-w%32,x[(w+64>>>9<<4)+14]=Math.floor(y/4294967296),x[(w+64>>>9<<4)+15]=y,h.sigBytes=x.length*4,this._process(),this._hash},clone:function(){var h=c.clone.call(this);return h._hash=this._hash.clone(),h}});a.SHA1=c._createHelper(f),a.HmacSHA1=c._createHmacHelper(f)})(),n.SHA1})})(Hs)),Hs.exports}var Vs={exports:{}},O2=Vs.exports,Dh;function Ld(){return Dh||(Dh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(O2,function(n){return(function(a){var i=n,s=i.lib,c=s.WordArray,u=s.Hasher,p=i.algo,f=[],h=[];(function(){function w(P){for(var b=a.sqrt(P),A=2;A<=b;A++)if(!(P%A))return!1;return!0}function E(P){return(P-(P|0))*4294967296|0}for(var C=2,S=0;S<64;)w(C)&&(S<8&&(f[S]=E(a.pow(C,1/2))),h[S]=E(a.pow(C,1/3)),S++),C++})();var x=[],y=p.SHA256=u.extend({_doReset:function(){this._hash=new c.init(f.slice(0))},_doProcessBlock:function(w,E){for(var C=this._hash.words,S=C[0],P=C[1],b=C[2],A=C[3],T=C[4],B=C[5],F=C[6],k=C[7],N=0;N<64;N++){if(N<16)x[N]=w[E+N]|0;else{var I=x[N-15],L=(I<<25|I>>>7)^(I<<14|I>>>18)^I>>>3,M=x[N-2],H=(M<<15|M>>>17)^(M<<13|M>>>19)^M>>>10;x[N]=L+x[N-7]+H+x[N-16]}var z=T&B^~T&F,Z=S&P^S&b^P&b,oe=(S<<30|S>>>2)^(S<<19|S>>>13)^(S<<10|S>>>22),re=(T<<26|T>>>6)^(T<<21|T>>>11)^(T<<7|T>>>25),se=k+re+z+h[N]+x[N],Y=oe+Z;k=F,F=B,B=T,T=A+se|0,A=b,b=P,P=S,S=se+Y|0}C[0]=C[0]+S|0,C[1]=C[1]+P|0,C[2]=C[2]+b|0,C[3]=C[3]+A|0,C[4]=C[4]+T|0,C[5]=C[5]+B|0,C[6]=C[6]+F|0,C[7]=C[7]+k|0},_doFinalize:function(){var w=this._data,E=w.words,C=this._nDataBytes*8,S=w.sigBytes*8;return E[S>>>5]|=128<<24-S%32,E[(S+64>>>9<<4)+14]=a.floor(C/4294967296),E[(S+64>>>9<<4)+15]=C,w.sigBytes=E.length*4,this._process(),this._hash},clone:function(){var w=u.clone.call(this);return w._hash=this._hash.clone(),w}});i.SHA256=u._createHelper(y),i.HmacSHA256=u._createHmacHelper(y)})(Math),n.SHA256})})(Vs)),Vs.exports}var Ws={exports:{}},M2=Ws.exports,Th;function $2(){return Th||(Th=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ld())})(M2,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=a.algo,u=c.SHA256,p=c.SHA224=u.extend({_doReset:function(){this._hash=new s.init([3238371032,914150663,812702999,4144912697,4290775857,1750603025,1694076839,3204075428])},_doFinalize:function(){var f=u._doFinalize.call(this);return f.sigBytes-=4,f}});a.SHA224=u._createHelper(p),a.HmacSHA224=u._createHmacHelper(p)})(),n.SHA224})})(Ws)),Ws.exports}var Gs={exports:{}},U2=Gs.exports,_h;function wg(){return _h||(_h=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ml())})(U2,function(n){return(function(){var a=n,i=a.lib,s=i.Hasher,c=a.x64,u=c.Word,p=c.WordArray,f=a.algo;function h(){return u.create.apply(u,arguments)}var x=[h(1116352408,3609767458),h(1899447441,602891725),h(3049323471,3964484399),h(3921009573,2173295548),h(961987163,4081628472),h(1508970993,3053834265),h(2453635748,2937671579),h(2870763221,3664609560),h(3624381080,2734883394),h(310598401,1164996542),h(607225278,1323610764),h(1426881987,3590304994),h(1925078388,4068182383),h(2162078206,991336113),h(2614888103,633803317),h(3248222580,3479774868),h(3835390401,2666613458),h(4022224774,944711139),h(264347078,2341262773),h(604807628,2007800933),h(770255983,1495990901),h(1249150122,1856431235),h(1555081692,3175218132),h(1996064986,2198950837),h(2554220882,3999719339),h(2821834349,766784016),h(2952996808,2566594879),h(3210313671,3203337956),h(3336571891,1034457026),h(3584528711,2466948901),h(113926993,3758326383),h(338241895,168717936),h(666307205,1188179964),h(773529912,1546045734),h(1294757372,1522805485),h(1396182291,2643833823),h(1695183700,2343527390),h(1986661051,1014477480),h(2177026350,1206759142),h(2456956037,344077627),h(2730485921,1290863460),h(2820302411,3158454273),h(3259730800,3505952657),h(3345764771,106217008),h(3516065817,3606008344),h(3600352804,1432725776),h(4094571909,1467031594),h(275423344,851169720),h(430227734,3100823752),h(506948616,1363258195),h(659060556,3750685593),h(883997877,3785050280),h(958139571,3318307427),h(1322822218,3812723403),h(1537002063,2003034995),h(1747873779,3602036899),h(1955562222,1575990012),h(2024104815,1125592928),h(2227730452,2716904306),h(2361852424,442776044),h(2428436474,593698344),h(2756734187,3733110249),h(3204031479,2999351573),h(3329325298,3815920427),h(3391569614,3928383900),h(3515267271,566280711),h(3940187606,3454069534),h(4118630271,4000239992),h(116418474,1914138554),h(174292421,2731055270),h(289380356,3203993006),h(460393269,320620315),h(685471733,587496836),h(852142971,1086792851),h(1017036298,365543100),h(1126000580,2618297676),h(1288033470,3409855158),h(1501505948,4234509866),h(1607167915,987167468),h(1816402316,1246189591)],y=[];(function(){for(var E=0;E<80;E++)y[E]=h()})();var w=f.SHA512=s.extend({_doReset:function(){this._hash=new p.init([new u.init(1779033703,4089235720),new u.init(3144134277,2227873595),new u.init(1013904242,4271175723),new u.init(2773480762,1595750129),new u.init(1359893119,2917565137),new u.init(2600822924,725511199),new u.init(528734635,4215389547),new u.init(1541459225,327033209)])},_doProcessBlock:function(E,C){for(var S=this._hash.words,P=S[0],b=S[1],A=S[2],T=S[3],B=S[4],F=S[5],k=S[6],N=S[7],I=P.high,L=P.low,M=b.high,H=b.low,z=A.high,Z=A.low,oe=T.high,re=T.low,se=B.high,Y=B.low,te=F.high,ee=F.low,_=k.high,$=k.low,G=N.high,K=N.low,ue=I,me=L,Le=M,he=H,Ve=z,ot=Z,Vt=oe,We=re,Ee=se,ge=Y,je=te,Ye=ee,tt=_,Dt=$,ke=G,xe=K,ye=0;ye<80;ye++){var Fe,He,Oe=y[ye];if(ye<16)He=Oe.high=E[C+ye*2]|0,Fe=Oe.low=E[C+ye*2+1]|0;else{var ct=y[ye-15],rt=ct.high,Rt=ct.low,Ke=(rt>>>1|Rt<<31)^(rt>>>8|Rt<<24)^rt>>>7,mt=(Rt>>>1|rt<<31)^(Rt>>>8|rt<<24)^(Rt>>>7|rt<<25),Tt=y[ye-2],Ft=Tt.high,cr=Tt.low,Po=(Ft>>>19|cr<<13)^(Ft<<3|cr>>>29)^Ft>>>6,Yt=(cr>>>19|Ft<<13)^(cr<<3|Ft>>>29)^(cr>>>6|Ft<<26),Zn=y[ye-7],mn=Zn.high,gn=Zn.low,xn=y[ye-16],vn=xn.high,Pr=xn.low;Fe=mt+gn,He=Ke+mn+(Fe>>>0<mt>>>0?1:0),Fe=Fe+Yt,He=He+Po+(Fe>>>0<Yt>>>0?1:0),Fe=Fe+Pr,He=He+vn+(Fe>>>0<Pr>>>0?1:0),Oe.high=He,Oe.low=Fe}var kr=Ee&je^~Ee&tt,ur=ge&Ye^~ge&Dt,ko=ue&Le^ue&Ve^Le&Ve,eo=me&he^me&ot^he&ot,Do=(ue>>>28|me<<4)^(ue<<30|me>>>2)^(ue<<25|me>>>7),yn=(me>>>28|ue<<4)^(me<<30|ue>>>2)^(me<<25|ue>>>7),Kr=(Ee>>>14|ge<<18)^(Ee>>>18|ge<<14)^(Ee<<23|ge>>>9),Or=(ge>>>14|Ee<<18)^(ge>>>18|Ee<<14)^(ge<<23|Ee>>>9),wn=x[ye],Xr=wn.high,En=wn.low,O=xe+Or,W=ke+Kr+(O>>>0<xe>>>0?1:0),O=O+ur,W=W+kr+(O>>>0<ur>>>0?1:0),O=O+En,W=W+Xr+(O>>>0<En>>>0?1:0),O=O+Fe,W=W+He+(O>>>0<Fe>>>0?1:0),Q=yn+eo,ae=Do+ko+(Q>>>0<yn>>>0?1:0);ke=tt,xe=Dt,tt=je,Dt=Ye,je=Ee,Ye=ge,ge=We+O|0,Ee=Vt+W+(ge>>>0<We>>>0?1:0)|0,Vt=Ve,We=ot,Ve=Le,ot=he,Le=ue,he=me,me=O+Q|0,ue=W+ae+(me>>>0<O>>>0?1:0)|0}L=P.low=L+me,P.high=I+ue+(L>>>0<me>>>0?1:0),H=b.low=H+he,b.high=M+Le+(H>>>0<he>>>0?1:0),Z=A.low=Z+ot,A.high=z+Ve+(Z>>>0<ot>>>0?1:0),re=T.low=re+We,T.high=oe+Vt+(re>>>0<We>>>0?1:0),Y=B.low=Y+ge,B.high=se+Ee+(Y>>>0<ge>>>0?1:0),ee=F.low=ee+Ye,F.high=te+je+(ee>>>0<Ye>>>0?1:0),$=k.low=$+Dt,k.high=_+tt+($>>>0<Dt>>>0?1:0),K=N.low=K+xe,N.high=G+ke+(K>>>0<xe>>>0?1:0)},_doFinalize:function(){var E=this._data,C=E.words,S=this._nDataBytes*8,P=E.sigBytes*8;C[P>>>5]|=128<<24-P%32,C[(P+128>>>10<<5)+30]=Math.floor(S/4294967296),C[(P+128>>>10<<5)+31]=S,E.sigBytes=C.length*4,this._process();var b=this._hash.toX32();return b},clone:function(){var E=s.clone.call(this);return E._hash=this._hash.clone(),E},blockSize:1024/32});a.SHA512=s._createHelper(w),a.HmacSHA512=s._createHmacHelper(w)})(),n.SHA512})})(Gs)),Gs.exports}var qs={exports:{}},z2=qs.exports,Lh;function H2(){return Lh||(Lh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ml(),wg())})(z2,function(n){return(function(){var a=n,i=a.x64,s=i.Word,c=i.WordArray,u=a.algo,p=u.SHA512,f=u.SHA384=p.extend({_doReset:function(){this._hash=new c.init([new s.init(3418070365,3238371032),new s.init(1654270250,914150663),new s.init(2438529370,812702999),new s.init(355462360,4144912697),new s.init(1731405415,4290775857),new s.init(2394180231,1750603025),new s.init(3675008525,1694076839),new s.init(1203062813,3204075428)])},_doFinalize:function(){var h=p._doFinalize.call(this);return h.sigBytes-=16,h}});a.SHA384=p._createHelper(f),a.HmacSHA384=p._createHmacHelper(f)})(),n.SHA384})})(qs)),qs.exports}var Ks={exports:{}},V2=Ks.exports,Fh;function W2(){return Fh||(Fh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ml())})(V2,function(n){return(function(a){var i=n,s=i.lib,c=s.WordArray,u=s.Hasher,p=i.x64,f=p.Word,h=i.algo,x=[],y=[],w=[];(function(){for(var S=1,P=0,b=0;b<24;b++){x[S+5*P]=(b+1)*(b+2)/2%64;var A=P%5,T=(2*S+3*P)%5;S=A,P=T}for(var S=0;S<5;S++)for(var P=0;P<5;P++)y[S+5*P]=P+(2*S+3*P)%5*5;for(var B=1,F=0;F<24;F++){for(var k=0,N=0,I=0;I<7;I++){if(B&1){var L=(1<<I)-1;L<32?N^=1<<L:k^=1<<L-32}B&128?B=B<<1^113:B<<=1}w[F]=f.create(k,N)}})();var E=[];(function(){for(var S=0;S<25;S++)E[S]=f.create()})();var C=h.SHA3=u.extend({cfg:u.cfg.extend({outputLength:512}),_doReset:function(){for(var S=this._state=[],P=0;P<25;P++)S[P]=new f.init;this.blockSize=(1600-2*this.cfg.outputLength)/32},_doProcessBlock:function(S,P){for(var b=this._state,A=this.blockSize/2,T=0;T<A;T++){var B=S[P+2*T],F=S[P+2*T+1];B=(B<<8|B>>>24)&16711935|(B<<24|B>>>8)&4278255360,F=(F<<8|F>>>24)&16711935|(F<<24|F>>>8)&4278255360;var k=b[T];k.high^=F,k.low^=B}for(var N=0;N<24;N++){for(var I=0;I<5;I++){for(var L=0,M=0,H=0;H<5;H++){var k=b[I+5*H];L^=k.high,M^=k.low}var z=E[I];z.high=L,z.low=M}for(var I=0;I<5;I++)for(var Z=E[(I+4)%5],oe=E[(I+1)%5],re=oe.high,se=oe.low,L=Z.high^(re<<1|se>>>31),M=Z.low^(se<<1|re>>>31),H=0;H<5;H++){var k=b[I+5*H];k.high^=L,k.low^=M}for(var Y=1;Y<25;Y++){var L,M,k=b[Y],te=k.high,ee=k.low,_=x[Y];_<32?(L=te<<_|ee>>>32-_,M=ee<<_|te>>>32-_):(L=ee<<_-32|te>>>64-_,M=te<<_-32|ee>>>64-_);var $=E[y[Y]];$.high=L,$.low=M}var G=E[0],K=b[0];G.high=K.high,G.low=K.low;for(var I=0;I<5;I++)for(var H=0;H<5;H++){var Y=I+5*H,k=b[Y],ue=E[Y],me=E[(I+1)%5+5*H],Le=E[(I+2)%5+5*H];k.high=ue.high^~me.high&Le.high,k.low=ue.low^~me.low&Le.low}var k=b[0],he=w[N];k.high^=he.high,k.low^=he.low}},_doFinalize:function(){var S=this._data,P=S.words;this._nDataBytes*8;var b=S.sigBytes*8,A=this.blockSize*32;P[b>>>5]|=1<<24-b%32,P[(a.ceil((b+1)/A)*A>>>5)-1]|=128,S.sigBytes=P.length*4,this._process();for(var T=this._state,B=this.cfg.outputLength/8,F=B/8,k=[],N=0;N<F;N++){var I=T[N],L=I.high,M=I.low;L=(L<<8|L>>>24)&16711935|(L<<24|L>>>8)&4278255360,M=(M<<8|M>>>24)&16711935|(M<<24|M>>>8)&4278255360,k.push(M),k.push(L)}return new c.init(k,B)},clone:function(){for(var S=u.clone.call(this),P=S._state=this._state.slice(0),b=0;b<25;b++)P[b]=P[b].clone();return S}});i.SHA3=u._createHelper(C),i.HmacSHA3=u._createHmacHelper(C)})(Math),n.SHA3})})(Ks)),Ks.exports}var Xs={exports:{}},G2=Xs.exports,Nh;function q2(){return Nh||(Nh=1,(function(r,t){(function(n,a){r.exports=a(Je())})(G2,function(n){/** @preserve
			(c) 2012 by Cédric Mesnil. All rights reserved.

			Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

			    - Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
			    - Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

			THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
			*/return(function(a){var i=n,s=i.lib,c=s.WordArray,u=s.Hasher,p=i.algo,f=c.create([0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,7,4,13,1,10,6,15,3,12,0,9,5,2,14,11,8,3,10,14,4,9,15,8,1,2,7,0,6,13,11,5,12,1,9,11,10,0,8,12,4,13,3,7,15,14,5,6,2,4,0,5,9,7,12,2,10,14,1,3,8,11,6,15,13]),h=c.create([5,14,7,0,9,2,11,4,13,6,15,8,1,10,3,12,6,11,3,7,0,13,5,10,14,15,8,12,4,9,1,2,15,5,1,3,7,14,6,9,11,8,12,2,10,0,4,13,8,6,4,1,3,11,15,0,5,12,2,13,9,7,10,14,12,15,10,4,1,5,8,7,6,2,13,14,0,3,9,11]),x=c.create([11,14,15,12,5,8,7,9,11,13,14,15,6,7,9,8,7,6,8,13,11,9,7,15,7,12,15,9,11,7,13,12,11,13,6,7,14,9,13,15,14,8,13,6,5,12,7,5,11,12,14,15,14,15,9,8,9,14,5,6,8,6,5,12,9,15,5,11,6,8,13,12,5,12,13,14,11,8,5,6]),y=c.create([8,9,9,11,13,15,15,5,7,7,8,11,14,14,12,6,9,13,15,7,12,8,9,11,7,7,12,7,6,15,13,11,9,7,15,11,8,6,6,14,12,13,5,14,13,13,7,5,15,5,8,11,14,14,6,14,6,9,12,9,12,5,15,8,8,5,12,9,12,5,14,6,8,13,6,5,15,13,11,11]),w=c.create([0,1518500249,1859775393,2400959708,2840853838]),E=c.create([1352829926,1548603684,1836072691,2053994217,0]),C=p.RIPEMD160=u.extend({_doReset:function(){this._hash=c.create([1732584193,4023233417,2562383102,271733878,3285377520])},_doProcessBlock:function(F,k){for(var N=0;N<16;N++){var I=k+N,L=F[I];F[I]=(L<<8|L>>>24)&16711935|(L<<24|L>>>8)&4278255360}var M=this._hash.words,H=w.words,z=E.words,Z=f.words,oe=h.words,re=x.words,se=y.words,Y,te,ee,_,$,G,K,ue,me,Le;G=Y=M[0],K=te=M[1],ue=ee=M[2],me=_=M[3],Le=$=M[4];for(var he,N=0;N<80;N+=1)he=Y+F[k+Z[N]]|0,N<16?he+=S(te,ee,_)+H[0]:N<32?he+=P(te,ee,_)+H[1]:N<48?he+=b(te,ee,_)+H[2]:N<64?he+=A(te,ee,_)+H[3]:he+=T(te,ee,_)+H[4],he=he|0,he=B(he,re[N]),he=he+$|0,Y=$,$=_,_=B(ee,10),ee=te,te=he,he=G+F[k+oe[N]]|0,N<16?he+=T(K,ue,me)+z[0]:N<32?he+=A(K,ue,me)+z[1]:N<48?he+=b(K,ue,me)+z[2]:N<64?he+=P(K,ue,me)+z[3]:he+=S(K,ue,me)+z[4],he=he|0,he=B(he,se[N]),he=he+Le|0,G=Le,Le=me,me=B(ue,10),ue=K,K=he;he=M[1]+ee+me|0,M[1]=M[2]+_+Le|0,M[2]=M[3]+$+G|0,M[3]=M[4]+Y+K|0,M[4]=M[0]+te+ue|0,M[0]=he},_doFinalize:function(){var F=this._data,k=F.words,N=this._nDataBytes*8,I=F.sigBytes*8;k[I>>>5]|=128<<24-I%32,k[(I+64>>>9<<4)+14]=(N<<8|N>>>24)&16711935|(N<<24|N>>>8)&4278255360,F.sigBytes=(k.length+1)*4,this._process();for(var L=this._hash,M=L.words,H=0;H<5;H++){var z=M[H];M[H]=(z<<8|z>>>24)&16711935|(z<<24|z>>>8)&4278255360}return L},clone:function(){var F=u.clone.call(this);return F._hash=this._hash.clone(),F}});function S(F,k,N){return F^k^N}function P(F,k,N){return F&k|~F&N}function b(F,k,N){return(F|~k)^N}function A(F,k,N){return F&N|k&~N}function T(F,k,N){return F^(k|~N)}function B(F,k){return F<<k|F>>>32-k}i.RIPEMD160=u._createHelper(C),i.HmacRIPEMD160=u._createHmacHelper(C)})(),n.RIPEMD160})})(Xs)),Xs.exports}var Ys={exports:{}},K2=Ys.exports,Ih;function Fd(){return Ih||(Ih=1,(function(r,t){(function(n,a){r.exports=a(Je())})(K2,function(n){(function(){var a=n,i=a.lib,s=i.Base,c=a.enc,u=c.Utf8,p=a.algo;p.HMAC=s.extend({init:function(f,h){f=this._hasher=new f.init,typeof h=="string"&&(h=u.parse(h));var x=f.blockSize,y=x*4;h.sigBytes>y&&(h=f.finalize(h)),h.clamp();for(var w=this._oKey=h.clone(),E=this._iKey=h.clone(),C=w.words,S=E.words,P=0;P<x;P++)C[P]^=1549556828,S[P]^=909522486;w.sigBytes=E.sigBytes=y,this.reset()},reset:function(){var f=this._hasher;f.reset(),f.update(this._iKey)},update:function(f){return this._hasher.update(f),this},finalize:function(f){var h=this._hasher,x=h.finalize(f);h.reset();var y=h.finalize(this._oKey.clone().concat(x));return y}})})()})})(Ys)),Ys.exports}var Qs={exports:{}},X2=Qs.exports,Bh;function Y2(){return Bh||(Bh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ld(),Fd())})(X2,function(n){return(function(){var a=n,i=a.lib,s=i.Base,c=i.WordArray,u=a.algo,p=u.SHA256,f=u.HMAC,h=u.PBKDF2=s.extend({cfg:s.extend({keySize:128/32,hasher:p,iterations:25e4}),init:function(x){this.cfg=this.cfg.extend(x)},compute:function(x,y){for(var w=this.cfg,E=f.create(w.hasher,x),C=c.create(),S=c.create([1]),P=C.words,b=S.words,A=w.keySize,T=w.iterations;P.length<A;){var B=E.update(y).finalize(S);E.reset();for(var F=B.words,k=F.length,N=B,I=1;I<T;I++){N=E.finalize(N),E.reset();for(var L=N.words,M=0;M<k;M++)F[M]^=L[M]}C.concat(B),b[0]++}return C.sigBytes=A*4,C}});a.PBKDF2=function(x,y,w){return h.create(w).compute(x,y)}})(),n.PBKDF2})})(Qs)),Qs.exports}var Js={exports:{}},Q2=Js.exports,jh;function Jn(){return jh||(jh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),yg(),Fd())})(Q2,function(n){return(function(){var a=n,i=a.lib,s=i.Base,c=i.WordArray,u=a.algo,p=u.MD5,f=u.EvpKDF=s.extend({cfg:s.extend({keySize:128/32,hasher:p,iterations:1}),init:function(h){this.cfg=this.cfg.extend(h)},compute:function(h,x){for(var y,w=this.cfg,E=w.hasher.create(),C=c.create(),S=C.words,P=w.keySize,b=w.iterations;S.length<P;){y&&E.update(y),y=E.update(h).finalize(x),E.reset();for(var A=1;A<b;A++)y=E.finalize(y),E.reset();C.concat(y)}return C.sigBytes=P*4,C}});a.EvpKDF=function(h,x,y){return f.create(y).compute(h,x)}})(),n.EvpKDF})})(Js)),Js.exports}var Zs={exports:{}},J2=Zs.exports,Oh;function $t(){return Oh||(Oh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Jn())})(J2,function(n){n.lib.Cipher||(function(a){var i=n,s=i.lib,c=s.Base,u=s.WordArray,p=s.BufferedBlockAlgorithm,f=i.enc;f.Utf8;var h=f.Base64,x=i.algo,y=x.EvpKDF,w=s.Cipher=p.extend({cfg:c.extend(),createEncryptor:function(L,M){return this.create(this._ENC_XFORM_MODE,L,M)},createDecryptor:function(L,M){return this.create(this._DEC_XFORM_MODE,L,M)},init:function(L,M,H){this.cfg=this.cfg.extend(H),this._xformMode=L,this._key=M,this.reset()},reset:function(){p.reset.call(this),this._doReset()},process:function(L){return this._append(L),this._process()},finalize:function(L){L&&this._append(L);var M=this._doFinalize();return M},keySize:128/32,ivSize:128/32,_ENC_XFORM_MODE:1,_DEC_XFORM_MODE:2,_createHelper:(function(){function L(M){return typeof M=="string"?I:F}return function(M){return{encrypt:function(H,z,Z){return L(z).encrypt(M,H,z,Z)},decrypt:function(H,z,Z){return L(z).decrypt(M,H,z,Z)}}}})()});s.StreamCipher=w.extend({_doFinalize:function(){var L=this._process(!0);return L},blockSize:1});var E=i.mode={},C=s.BlockCipherMode=c.extend({createEncryptor:function(L,M){return this.Encryptor.create(L,M)},createDecryptor:function(L,M){return this.Decryptor.create(L,M)},init:function(L,M){this._cipher=L,this._iv=M}}),S=E.CBC=(function(){var L=C.extend();L.Encryptor=L.extend({processBlock:function(H,z){var Z=this._cipher,oe=Z.blockSize;M.call(this,H,z,oe),Z.encryptBlock(H,z),this._prevBlock=H.slice(z,z+oe)}}),L.Decryptor=L.extend({processBlock:function(H,z){var Z=this._cipher,oe=Z.blockSize,re=H.slice(z,z+oe);Z.decryptBlock(H,z),M.call(this,H,z,oe),this._prevBlock=re}});function M(H,z,Z){var oe,re=this._iv;re?(oe=re,this._iv=a):oe=this._prevBlock;for(var se=0;se<Z;se++)H[z+se]^=oe[se]}return L})(),P=i.pad={},b=P.Pkcs7={pad:function(L,M){for(var H=M*4,z=H-L.sigBytes%H,Z=z<<24|z<<16|z<<8|z,oe=[],re=0;re<z;re+=4)oe.push(Z);var se=u.create(oe,z);L.concat(se)},unpad:function(L){var M=L.words[L.sigBytes-1>>>2]&255;L.sigBytes-=M}};s.BlockCipher=w.extend({cfg:w.cfg.extend({mode:S,padding:b}),reset:function(){var L;w.reset.call(this);var M=this.cfg,H=M.iv,z=M.mode;this._xformMode==this._ENC_XFORM_MODE?L=z.createEncryptor:(L=z.createDecryptor,this._minBufferSize=1),this._mode&&this._mode.__creator==L?this._mode.init(this,H&&H.words):(this._mode=L.call(z,this,H&&H.words),this._mode.__creator=L)},_doProcessBlock:function(L,M){this._mode.processBlock(L,M)},_doFinalize:function(){var L,M=this.cfg.padding;return this._xformMode==this._ENC_XFORM_MODE?(M.pad(this._data,this.blockSize),L=this._process(!0)):(L=this._process(!0),M.unpad(L)),L},blockSize:128/32});var A=s.CipherParams=c.extend({init:function(L){this.mixIn(L)},toString:function(L){return(L||this.formatter).stringify(this)}}),T=i.format={},B=T.OpenSSL={stringify:function(L){var M,H=L.ciphertext,z=L.salt;return z?M=u.create([1398893684,1701076831]).concat(z).concat(H):M=H,M.toString(h)},parse:function(L){var M,H=h.parse(L),z=H.words;return z[0]==1398893684&&z[1]==1701076831&&(M=u.create(z.slice(2,4)),z.splice(0,4),H.sigBytes-=16),A.create({ciphertext:H,salt:M})}},F=s.SerializableCipher=c.extend({cfg:c.extend({format:B}),encrypt:function(L,M,H,z){z=this.cfg.extend(z);var Z=L.createEncryptor(H,z),oe=Z.finalize(M),re=Z.cfg;return A.create({ciphertext:oe,key:H,iv:re.iv,algorithm:L,mode:re.mode,padding:re.padding,blockSize:L.blockSize,formatter:z.format})},decrypt:function(L,M,H,z){z=this.cfg.extend(z),M=this._parse(M,z.format);var Z=L.createDecryptor(H,z).finalize(M.ciphertext);return Z},_parse:function(L,M){return typeof L=="string"?M.parse(L,this):L}}),k=i.kdf={},N=k.OpenSSL={execute:function(L,M,H,z,Z){if(z||(z=u.random(64/8)),Z)var oe=y.create({keySize:M+H,hasher:Z}).compute(L,z);else var oe=y.create({keySize:M+H}).compute(L,z);var re=u.create(oe.words.slice(M),H*4);return oe.sigBytes=M*4,A.create({key:oe,iv:re,salt:z})}},I=s.PasswordBasedCipher=F.extend({cfg:F.cfg.extend({kdf:N}),encrypt:function(L,M,H,z){z=this.cfg.extend(z);var Z=z.kdf.execute(H,L.keySize,L.ivSize,z.salt,z.hasher);z.iv=Z.iv;var oe=F.encrypt.call(this,L,M,Z.key,z);return oe.mixIn(Z),oe},decrypt:function(L,M,H,z){z=this.cfg.extend(z),M=this._parse(M,z.format);var Z=z.kdf.execute(H,L.keySize,L.ivSize,M.salt,z.hasher);z.iv=Z.iv;var oe=F.decrypt.call(this,L,M,Z.key,z);return oe}})})()})})(Zs)),Zs.exports}var el={exports:{}},Z2=el.exports,Mh;function eE(){return Mh||(Mh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(Z2,function(n){return n.mode.CFB=(function(){var a=n.lib.BlockCipherMode.extend();a.Encryptor=a.extend({processBlock:function(s,c){var u=this._cipher,p=u.blockSize;i.call(this,s,c,p,u),this._prevBlock=s.slice(c,c+p)}}),a.Decryptor=a.extend({processBlock:function(s,c){var u=this._cipher,p=u.blockSize,f=s.slice(c,c+p);i.call(this,s,c,p,u),this._prevBlock=f}});function i(s,c,u,p){var f,h=this._iv;h?(f=h.slice(0),this._iv=void 0):f=this._prevBlock,p.encryptBlock(f,0);for(var x=0;x<u;x++)s[c+x]^=f[x]}return a})(),n.mode.CFB})})(el)),el.exports}var tl={exports:{}},tE=tl.exports,$h;function rE(){return $h||($h=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(tE,function(n){return n.mode.CTR=(function(){var a=n.lib.BlockCipherMode.extend(),i=a.Encryptor=a.extend({processBlock:function(s,c){var u=this._cipher,p=u.blockSize,f=this._iv,h=this._counter;f&&(h=this._counter=f.slice(0),this._iv=void 0);var x=h.slice(0);u.encryptBlock(x,0),h[p-1]=h[p-1]+1|0;for(var y=0;y<p;y++)s[c+y]^=x[y]}});return a.Decryptor=i,a})(),n.mode.CTR})})(tl)),tl.exports}var rl={exports:{}},nE=rl.exports,Uh;function oE(){return Uh||(Uh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(nE,function(n){/** @preserve
 * Counter block mode compatible with  Dr Brian Gladman fileenc.c
 * derived from CryptoJS.mode.CTR
 * Jan Hruby jhruby.web@gmail.com
 */return n.mode.CTRGladman=(function(){var a=n.lib.BlockCipherMode.extend();function i(u){if((u>>24&255)===255){var p=u>>16&255,f=u>>8&255,h=u&255;p===255?(p=0,f===255?(f=0,h===255?h=0:++h):++f):++p,u=0,u+=p<<16,u+=f<<8,u+=h}else u+=1<<24;return u}function s(u){return(u[0]=i(u[0]))===0&&(u[1]=i(u[1])),u}var c=a.Encryptor=a.extend({processBlock:function(u,p){var f=this._cipher,h=f.blockSize,x=this._iv,y=this._counter;x&&(y=this._counter=x.slice(0),this._iv=void 0),s(y);var w=y.slice(0);f.encryptBlock(w,0);for(var E=0;E<h;E++)u[p+E]^=w[E]}});return a.Decryptor=c,a})(),n.mode.CTRGladman})})(rl)),rl.exports}var nl={exports:{}},aE=nl.exports,zh;function iE(){return zh||(zh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(aE,function(n){return n.mode.OFB=(function(){var a=n.lib.BlockCipherMode.extend(),i=a.Encryptor=a.extend({processBlock:function(s,c){var u=this._cipher,p=u.blockSize,f=this._iv,h=this._keystream;f&&(h=this._keystream=f.slice(0),this._iv=void 0),u.encryptBlock(h,0);for(var x=0;x<p;x++)s[c+x]^=h[x]}});return a.Decryptor=i,a})(),n.mode.OFB})})(nl)),nl.exports}var ol={exports:{}},sE=ol.exports,Hh;function lE(){return Hh||(Hh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(sE,function(n){return n.mode.ECB=(function(){var a=n.lib.BlockCipherMode.extend();return a.Encryptor=a.extend({processBlock:function(i,s){this._cipher.encryptBlock(i,s)}}),a.Decryptor=a.extend({processBlock:function(i,s){this._cipher.decryptBlock(i,s)}}),a})(),n.mode.ECB})})(ol)),ol.exports}var al={exports:{}},cE=al.exports,Vh;function uE(){return Vh||(Vh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(cE,function(n){return n.pad.AnsiX923={pad:function(a,i){var s=a.sigBytes,c=i*4,u=c-s%c,p=s+u-1;a.clamp(),a.words[p>>>2]|=u<<24-p%4*8,a.sigBytes+=u},unpad:function(a){var i=a.words[a.sigBytes-1>>>2]&255;a.sigBytes-=i}},n.pad.Ansix923})})(al)),al.exports}var il={exports:{}},dE=il.exports,Wh;function fE(){return Wh||(Wh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(dE,function(n){return n.pad.Iso10126={pad:function(a,i){var s=i*4,c=s-a.sigBytes%s;a.concat(n.lib.WordArray.random(c-1)).concat(n.lib.WordArray.create([c<<24],1))},unpad:function(a){var i=a.words[a.sigBytes-1>>>2]&255;a.sigBytes-=i}},n.pad.Iso10126})})(il)),il.exports}var sl={exports:{}},pE=sl.exports,Gh;function hE(){return Gh||(Gh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(pE,function(n){return n.pad.Iso97971={pad:function(a,i){a.concat(n.lib.WordArray.create([2147483648],1)),n.pad.ZeroPadding.pad(a,i)},unpad:function(a){n.pad.ZeroPadding.unpad(a),a.sigBytes--}},n.pad.Iso97971})})(sl)),sl.exports}var ll={exports:{}},mE=ll.exports,qh;function gE(){return qh||(qh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(mE,function(n){return n.pad.ZeroPadding={pad:function(a,i){var s=i*4;a.clamp(),a.sigBytes+=s-(a.sigBytes%s||s)},unpad:function(a){for(var i=a.words,s=a.sigBytes-1,s=a.sigBytes-1;s>=0;s--)if(i[s>>>2]>>>24-s%4*8&255){a.sigBytes=s+1;break}}},n.pad.ZeroPadding})})(ll)),ll.exports}var cl={exports:{}},xE=cl.exports,Kh;function vE(){return Kh||(Kh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(xE,function(n){return n.pad.NoPadding={pad:function(){},unpad:function(){}},n.pad.NoPadding})})(cl)),cl.exports}var ul={exports:{}},yE=ul.exports,Xh;function wE(){return Xh||(Xh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),$t())})(yE,function(n){return(function(a){var i=n,s=i.lib,c=s.CipherParams,u=i.enc,p=u.Hex,f=i.format;f.Hex={stringify:function(h){return h.ciphertext.toString(p)},parse:function(h){var x=p.parse(h);return c.create({ciphertext:x})}}})(),n.format.Hex})})(ul)),ul.exports}var dl={exports:{}},EE=dl.exports,Yh;function CE(){return Yh||(Yh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(EE,function(n){return(function(){var a=n,i=a.lib,s=i.BlockCipher,c=a.algo,u=[],p=[],f=[],h=[],x=[],y=[],w=[],E=[],C=[],S=[];(function(){for(var A=[],T=0;T<256;T++)T<128?A[T]=T<<1:A[T]=T<<1^283;for(var B=0,F=0,T=0;T<256;T++){var k=F^F<<1^F<<2^F<<3^F<<4;k=k>>>8^k&255^99,u[B]=k,p[k]=B;var N=A[B],I=A[N],L=A[I],M=A[k]*257^k*16843008;f[B]=M<<24|M>>>8,h[B]=M<<16|M>>>16,x[B]=M<<8|M>>>24,y[B]=M;var M=L*16843009^I*65537^N*257^B*16843008;w[k]=M<<24|M>>>8,E[k]=M<<16|M>>>16,C[k]=M<<8|M>>>24,S[k]=M,B?(B=N^A[A[A[L^N]]],F^=A[A[F]]):B=F=1}})();var P=[0,1,2,4,8,16,32,64,128,27,54],b=c.AES=s.extend({_doReset:function(){var A;if(!(this._nRounds&&this._keyPriorReset===this._key)){for(var T=this._keyPriorReset=this._key,B=T.words,F=T.sigBytes/4,k=this._nRounds=F+6,N=(k+1)*4,I=this._keySchedule=[],L=0;L<N;L++)L<F?I[L]=B[L]:(A=I[L-1],L%F?F>6&&L%F==4&&(A=u[A>>>24]<<24|u[A>>>16&255]<<16|u[A>>>8&255]<<8|u[A&255]):(A=A<<8|A>>>24,A=u[A>>>24]<<24|u[A>>>16&255]<<16|u[A>>>8&255]<<8|u[A&255],A^=P[L/F|0]<<24),I[L]=I[L-F]^A);for(var M=this._invKeySchedule=[],H=0;H<N;H++){var L=N-H;if(H%4)var A=I[L];else var A=I[L-4];H<4||L<=4?M[H]=A:M[H]=w[u[A>>>24]]^E[u[A>>>16&255]]^C[u[A>>>8&255]]^S[u[A&255]]}}},encryptBlock:function(A,T){this._doCryptBlock(A,T,this._keySchedule,f,h,x,y,u)},decryptBlock:function(A,T){var B=A[T+1];A[T+1]=A[T+3],A[T+3]=B,this._doCryptBlock(A,T,this._invKeySchedule,w,E,C,S,p);var B=A[T+1];A[T+1]=A[T+3],A[T+3]=B},_doCryptBlock:function(A,T,B,F,k,N,I,L){for(var M=this._nRounds,H=A[T]^B[0],z=A[T+1]^B[1],Z=A[T+2]^B[2],oe=A[T+3]^B[3],re=4,se=1;se<M;se++){var Y=F[H>>>24]^k[z>>>16&255]^N[Z>>>8&255]^I[oe&255]^B[re++],te=F[z>>>24]^k[Z>>>16&255]^N[oe>>>8&255]^I[H&255]^B[re++],ee=F[Z>>>24]^k[oe>>>16&255]^N[H>>>8&255]^I[z&255]^B[re++],_=F[oe>>>24]^k[H>>>16&255]^N[z>>>8&255]^I[Z&255]^B[re++];H=Y,z=te,Z=ee,oe=_}var Y=(L[H>>>24]<<24|L[z>>>16&255]<<16|L[Z>>>8&255]<<8|L[oe&255])^B[re++],te=(L[z>>>24]<<24|L[Z>>>16&255]<<16|L[oe>>>8&255]<<8|L[H&255])^B[re++],ee=(L[Z>>>24]<<24|L[oe>>>16&255]<<16|L[H>>>8&255]<<8|L[z&255])^B[re++],_=(L[oe>>>24]<<24|L[H>>>16&255]<<16|L[z>>>8&255]<<8|L[Z&255])^B[re++];A[T]=Y,A[T+1]=te,A[T+2]=ee,A[T+3]=_},keySize:256/32});a.AES=s._createHelper(b)})(),n.AES})})(dl)),dl.exports}var fl={exports:{}},bE=fl.exports,Qh;function SE(){return Qh||(Qh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(bE,function(n){return(function(){var a=n,i=a.lib,s=i.WordArray,c=i.BlockCipher,u=a.algo,p=[57,49,41,33,25,17,9,1,58,50,42,34,26,18,10,2,59,51,43,35,27,19,11,3,60,52,44,36,63,55,47,39,31,23,15,7,62,54,46,38,30,22,14,6,61,53,45,37,29,21,13,5,28,20,12,4],f=[14,17,11,24,1,5,3,28,15,6,21,10,23,19,12,4,26,8,16,7,27,20,13,2,41,52,31,37,47,55,30,40,51,45,33,48,44,49,39,56,34,53,46,42,50,36,29,32],h=[1,2,4,6,8,10,12,14,15,17,19,21,23,25,27,28],x=[{0:8421888,268435456:32768,536870912:8421378,805306368:2,1073741824:512,1342177280:8421890,1610612736:8389122,1879048192:8388608,2147483648:514,2415919104:8389120,2684354560:33280,2952790016:8421376,3221225472:32770,3489660928:8388610,3758096384:0,4026531840:33282,134217728:0,402653184:8421890,671088640:33282,939524096:32768,1207959552:8421888,1476395008:512,1744830464:8421378,2013265920:2,2281701376:8389120,2550136832:33280,2818572288:8421376,3087007744:8389122,3355443200:8388610,3623878656:32770,3892314112:514,4160749568:8388608,1:32768,268435457:2,536870913:8421888,805306369:8388608,1073741825:8421378,1342177281:33280,1610612737:512,1879048193:8389122,2147483649:8421890,2415919105:8421376,2684354561:8388610,2952790017:33282,3221225473:514,3489660929:8389120,3758096385:32770,4026531841:0,134217729:8421890,402653185:8421376,671088641:8388608,939524097:512,1207959553:32768,1476395009:8388610,1744830465:2,2013265921:33282,2281701377:32770,2550136833:8389122,2818572289:514,3087007745:8421888,3355443201:8389120,3623878657:0,3892314113:33280,4160749569:8421378},{0:1074282512,16777216:16384,33554432:524288,50331648:1074266128,67108864:1073741840,83886080:1074282496,100663296:1073758208,117440512:16,134217728:540672,150994944:1073758224,167772160:1073741824,184549376:540688,201326592:524304,218103808:0,234881024:16400,251658240:1074266112,8388608:1073758208,25165824:540688,41943040:16,58720256:1073758224,75497472:1074282512,92274688:1073741824,109051904:524288,125829120:1074266128,142606336:524304,159383552:0,176160768:16384,192937984:1074266112,209715200:1073741840,226492416:540672,243269632:1074282496,260046848:16400,268435456:0,285212672:1074266128,301989888:1073758224,318767104:1074282496,335544320:1074266112,352321536:16,369098752:540688,385875968:16384,402653184:16400,419430400:524288,436207616:524304,452984832:1073741840,469762048:540672,486539264:1073758208,503316480:1073741824,520093696:1074282512,276824064:540688,293601280:524288,310378496:1074266112,327155712:16384,343932928:1073758208,360710144:1074282512,377487360:16,394264576:1073741824,411041792:1074282496,427819008:1073741840,444596224:1073758224,461373440:524304,478150656:0,494927872:16400,511705088:1074266128,528482304:540672},{0:260,1048576:0,2097152:67109120,3145728:65796,4194304:65540,5242880:67108868,6291456:67174660,7340032:67174400,8388608:67108864,9437184:67174656,10485760:65792,11534336:67174404,12582912:67109124,13631488:65536,14680064:4,15728640:256,524288:67174656,1572864:67174404,2621440:0,3670016:67109120,4718592:67108868,5767168:65536,6815744:65540,7864320:260,8912896:4,9961472:256,11010048:67174400,12058624:65796,13107200:65792,14155776:67109124,15204352:67174660,16252928:67108864,16777216:67174656,17825792:65540,18874368:65536,19922944:67109120,20971520:256,22020096:67174660,23068672:67108868,24117248:0,25165824:67109124,26214400:67108864,27262976:4,28311552:65792,29360128:67174400,30408704:260,31457280:65796,32505856:67174404,17301504:67108864,18350080:260,19398656:67174656,20447232:0,21495808:65540,22544384:67109120,23592960:256,24641536:67174404,25690112:65536,26738688:67174660,27787264:65796,28835840:67108868,29884416:67109124,30932992:67174400,31981568:4,33030144:65792},{0:2151682048,65536:2147487808,131072:4198464,196608:2151677952,262144:0,327680:4198400,393216:2147483712,458752:4194368,524288:2147483648,589824:4194304,655360:64,720896:2147487744,786432:2151678016,851968:4160,917504:4096,983040:2151682112,32768:2147487808,98304:64,163840:2151678016,229376:2147487744,294912:4198400,360448:2151682112,425984:0,491520:2151677952,557056:4096,622592:2151682048,688128:4194304,753664:4160,819200:2147483648,884736:4194368,950272:4198464,1015808:2147483712,1048576:4194368,1114112:4198400,1179648:2147483712,1245184:0,1310720:4160,1376256:2151678016,1441792:2151682048,1507328:2147487808,1572864:2151682112,1638400:2147483648,1703936:2151677952,1769472:4198464,1835008:2147487744,1900544:4194304,1966080:64,2031616:4096,1081344:2151677952,1146880:2151682112,1212416:0,1277952:4198400,1343488:4194368,1409024:2147483648,1474560:2147487808,1540096:64,1605632:2147483712,1671168:4096,1736704:2147487744,1802240:2151678016,1867776:4160,1933312:2151682048,1998848:4194304,2064384:4198464},{0:128,4096:17039360,8192:262144,12288:536870912,16384:537133184,20480:16777344,24576:553648256,28672:262272,32768:16777216,36864:537133056,40960:536871040,45056:553910400,49152:553910272,53248:0,57344:17039488,61440:553648128,2048:17039488,6144:553648256,10240:128,14336:17039360,18432:262144,22528:537133184,26624:553910272,30720:536870912,34816:537133056,38912:0,43008:553910400,47104:16777344,51200:536871040,55296:553648128,59392:16777216,63488:262272,65536:262144,69632:128,73728:536870912,77824:553648256,81920:16777344,86016:553910272,90112:537133184,94208:16777216,98304:553910400,102400:553648128,106496:17039360,110592:537133056,114688:262272,118784:536871040,122880:0,126976:17039488,67584:553648256,71680:16777216,75776:17039360,79872:537133184,83968:536870912,88064:17039488,92160:128,96256:553910272,100352:262272,104448:553910400,108544:0,112640:553648128,116736:16777344,120832:262144,124928:537133056,129024:536871040},{0:268435464,256:8192,512:270532608,768:270540808,1024:268443648,1280:2097152,1536:2097160,1792:268435456,2048:0,2304:268443656,2560:2105344,2816:8,3072:270532616,3328:2105352,3584:8200,3840:270540800,128:270532608,384:270540808,640:8,896:2097152,1152:2105352,1408:268435464,1664:268443648,1920:8200,2176:2097160,2432:8192,2688:268443656,2944:270532616,3200:0,3456:270540800,3712:2105344,3968:268435456,4096:268443648,4352:270532616,4608:270540808,4864:8200,5120:2097152,5376:268435456,5632:268435464,5888:2105344,6144:2105352,6400:0,6656:8,6912:270532608,7168:8192,7424:268443656,7680:270540800,7936:2097160,4224:8,4480:2105344,4736:2097152,4992:268435464,5248:268443648,5504:8200,5760:270540808,6016:270532608,6272:270540800,6528:270532616,6784:8192,7040:2105352,7296:2097160,7552:0,7808:268435456,8064:268443656},{0:1048576,16:33555457,32:1024,48:1049601,64:34604033,80:0,96:1,112:34603009,128:33555456,144:1048577,160:33554433,176:34604032,192:34603008,208:1025,224:1049600,240:33554432,8:34603009,24:0,40:33555457,56:34604032,72:1048576,88:33554433,104:33554432,120:1025,136:1049601,152:33555456,168:34603008,184:1048577,200:1024,216:34604033,232:1,248:1049600,256:33554432,272:1048576,288:33555457,304:34603009,320:1048577,336:33555456,352:34604032,368:1049601,384:1025,400:34604033,416:1049600,432:1,448:0,464:34603008,480:33554433,496:1024,264:1049600,280:33555457,296:34603009,312:1,328:33554432,344:1048576,360:1025,376:34604032,392:33554433,408:34603008,424:0,440:34604033,456:1049601,472:1024,488:33555456,504:1048577},{0:134219808,1:131072,2:134217728,3:32,4:131104,5:134350880,6:134350848,7:2048,8:134348800,9:134219776,10:133120,11:134348832,12:2080,13:0,14:134217760,15:133152,2147483648:2048,2147483649:134350880,2147483650:134219808,2147483651:134217728,2147483652:134348800,2147483653:133120,2147483654:133152,2147483655:32,2147483656:134217760,2147483657:2080,2147483658:131104,2147483659:134350848,2147483660:0,2147483661:134348832,2147483662:134219776,2147483663:131072,16:133152,17:134350848,18:32,19:2048,20:134219776,21:134217760,22:134348832,23:131072,24:0,25:131104,26:134348800,27:134219808,28:134350880,29:133120,30:2080,31:134217728,2147483664:131072,2147483665:2048,2147483666:134348832,2147483667:133152,2147483668:32,2147483669:134348800,2147483670:134217728,2147483671:134219808,2147483672:134350880,2147483673:134217760,2147483674:134219776,2147483675:0,2147483676:133120,2147483677:2080,2147483678:131104,2147483679:134350848}],y=[4160749569,528482304,33030144,2064384,129024,8064,504,2147483679],w=u.DES=c.extend({_doReset:function(){for(var P=this._key,b=P.words,A=[],T=0;T<56;T++){var B=p[T]-1;A[T]=b[B>>>5]>>>31-B%32&1}for(var F=this._subKeys=[],k=0;k<16;k++){for(var N=F[k]=[],I=h[k],T=0;T<24;T++)N[T/6|0]|=A[(f[T]-1+I)%28]<<31-T%6,N[4+(T/6|0)]|=A[28+(f[T+24]-1+I)%28]<<31-T%6;N[0]=N[0]<<1|N[0]>>>31;for(var T=1;T<7;T++)N[T]=N[T]>>>(T-1)*4+3;N[7]=N[7]<<5|N[7]>>>27}for(var L=this._invSubKeys=[],T=0;T<16;T++)L[T]=F[15-T]},encryptBlock:function(P,b){this._doCryptBlock(P,b,this._subKeys)},decryptBlock:function(P,b){this._doCryptBlock(P,b,this._invSubKeys)},_doCryptBlock:function(P,b,A){this._lBlock=P[b],this._rBlock=P[b+1],E.call(this,4,252645135),E.call(this,16,65535),C.call(this,2,858993459),C.call(this,8,16711935),E.call(this,1,1431655765);for(var T=0;T<16;T++){for(var B=A[T],F=this._lBlock,k=this._rBlock,N=0,I=0;I<8;I++)N|=x[I][((k^B[I])&y[I])>>>0];this._lBlock=k,this._rBlock=F^N}var L=this._lBlock;this._lBlock=this._rBlock,this._rBlock=L,E.call(this,1,1431655765),C.call(this,8,16711935),C.call(this,2,858993459),E.call(this,16,65535),E.call(this,4,252645135),P[b]=this._lBlock,P[b+1]=this._rBlock},keySize:64/32,ivSize:64/32,blockSize:64/32});function E(P,b){var A=(this._lBlock>>>P^this._rBlock)&b;this._rBlock^=A,this._lBlock^=A<<P}function C(P,b){var A=(this._rBlock>>>P^this._lBlock)&b;this._lBlock^=A,this._rBlock^=A<<P}a.DES=c._createHelper(w);var S=u.TripleDES=c.extend({_doReset:function(){var P=this._key,b=P.words;if(b.length!==2&&b.length!==4&&b.length<6)throw new Error("Invalid key length - 3DES requires the key length to be 64, 128, 192 or >192.");var A=b.slice(0,2),T=b.length<4?b.slice(0,2):b.slice(2,4),B=b.length<6?b.slice(0,2):b.slice(4,6);this._des1=w.createEncryptor(s.create(A)),this._des2=w.createEncryptor(s.create(T)),this._des3=w.createEncryptor(s.create(B))},encryptBlock:function(P,b){this._des1.encryptBlock(P,b),this._des2.decryptBlock(P,b),this._des3.encryptBlock(P,b)},decryptBlock:function(P,b){this._des3.decryptBlock(P,b),this._des2.encryptBlock(P,b),this._des1.decryptBlock(P,b)},keySize:192/32,ivSize:64/32,blockSize:64/32});a.TripleDES=c._createHelper(S)})(),n.TripleDES})})(fl)),fl.exports}var pl={exports:{}},RE=pl.exports,Jh;function AE(){return Jh||(Jh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(RE,function(n){return(function(){var a=n,i=a.lib,s=i.StreamCipher,c=a.algo,u=c.RC4=s.extend({_doReset:function(){for(var h=this._key,x=h.words,y=h.sigBytes,w=this._S=[],E=0;E<256;E++)w[E]=E;for(var E=0,C=0;E<256;E++){var S=E%y,P=x[S>>>2]>>>24-S%4*8&255;C=(C+w[E]+P)%256;var b=w[E];w[E]=w[C],w[C]=b}this._i=this._j=0},_doProcessBlock:function(h,x){h[x]^=p.call(this)},keySize:256/32,ivSize:0});function p(){for(var h=this._S,x=this._i,y=this._j,w=0,E=0;E<4;E++){x=(x+1)%256,y=(y+h[x])%256;var C=h[x];h[x]=h[y],h[y]=C,w|=h[(h[x]+h[y])%256]<<24-E*8}return this._i=x,this._j=y,w}a.RC4=s._createHelper(u);var f=c.RC4Drop=u.extend({cfg:u.cfg.extend({drop:192}),_doReset:function(){u._doReset.call(this);for(var h=this.cfg.drop;h>0;h--)p.call(this)}});a.RC4Drop=s._createHelper(f)})(),n.RC4})})(pl)),pl.exports}var hl={exports:{}},PE=hl.exports,Zh;function kE(){return Zh||(Zh=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(PE,function(n){return(function(){var a=n,i=a.lib,s=i.StreamCipher,c=a.algo,u=[],p=[],f=[],h=c.Rabbit=s.extend({_doReset:function(){for(var y=this._key.words,w=this.cfg.iv,E=0;E<4;E++)y[E]=(y[E]<<8|y[E]>>>24)&16711935|(y[E]<<24|y[E]>>>8)&4278255360;var C=this._X=[y[0],y[3]<<16|y[2]>>>16,y[1],y[0]<<16|y[3]>>>16,y[2],y[1]<<16|y[0]>>>16,y[3],y[2]<<16|y[1]>>>16],S=this._C=[y[2]<<16|y[2]>>>16,y[0]&4294901760|y[1]&65535,y[3]<<16|y[3]>>>16,y[1]&4294901760|y[2]&65535,y[0]<<16|y[0]>>>16,y[2]&4294901760|y[3]&65535,y[1]<<16|y[1]>>>16,y[3]&4294901760|y[0]&65535];this._b=0;for(var E=0;E<4;E++)x.call(this);for(var E=0;E<8;E++)S[E]^=C[E+4&7];if(w){var P=w.words,b=P[0],A=P[1],T=(b<<8|b>>>24)&16711935|(b<<24|b>>>8)&4278255360,B=(A<<8|A>>>24)&16711935|(A<<24|A>>>8)&4278255360,F=T>>>16|B&4294901760,k=B<<16|T&65535;S[0]^=T,S[1]^=F,S[2]^=B,S[3]^=k,S[4]^=T,S[5]^=F,S[6]^=B,S[7]^=k;for(var E=0;E<4;E++)x.call(this)}},_doProcessBlock:function(y,w){var E=this._X;x.call(this),u[0]=E[0]^E[5]>>>16^E[3]<<16,u[1]=E[2]^E[7]>>>16^E[5]<<16,u[2]=E[4]^E[1]>>>16^E[7]<<16,u[3]=E[6]^E[3]>>>16^E[1]<<16;for(var C=0;C<4;C++)u[C]=(u[C]<<8|u[C]>>>24)&16711935|(u[C]<<24|u[C]>>>8)&4278255360,y[w+C]^=u[C]},blockSize:128/32,ivSize:64/32});function x(){for(var y=this._X,w=this._C,E=0;E<8;E++)p[E]=w[E];w[0]=w[0]+1295307597+this._b|0,w[1]=w[1]+3545052371+(w[0]>>>0<p[0]>>>0?1:0)|0,w[2]=w[2]+886263092+(w[1]>>>0<p[1]>>>0?1:0)|0,w[3]=w[3]+1295307597+(w[2]>>>0<p[2]>>>0?1:0)|0,w[4]=w[4]+3545052371+(w[3]>>>0<p[3]>>>0?1:0)|0,w[5]=w[5]+886263092+(w[4]>>>0<p[4]>>>0?1:0)|0,w[6]=w[6]+1295307597+(w[5]>>>0<p[5]>>>0?1:0)|0,w[7]=w[7]+3545052371+(w[6]>>>0<p[6]>>>0?1:0)|0,this._b=w[7]>>>0<p[7]>>>0?1:0;for(var E=0;E<8;E++){var C=y[E]+w[E],S=C&65535,P=C>>>16,b=((S*S>>>17)+S*P>>>15)+P*P,A=((C&4294901760)*C|0)+((C&65535)*C|0);f[E]=b^A}y[0]=f[0]+(f[7]<<16|f[7]>>>16)+(f[6]<<16|f[6]>>>16)|0,y[1]=f[1]+(f[0]<<8|f[0]>>>24)+f[7]|0,y[2]=f[2]+(f[1]<<16|f[1]>>>16)+(f[0]<<16|f[0]>>>16)|0,y[3]=f[3]+(f[2]<<8|f[2]>>>24)+f[1]|0,y[4]=f[4]+(f[3]<<16|f[3]>>>16)+(f[2]<<16|f[2]>>>16)|0,y[5]=f[5]+(f[4]<<8|f[4]>>>24)+f[3]|0,y[6]=f[6]+(f[5]<<16|f[5]>>>16)+(f[4]<<16|f[4]>>>16)|0,y[7]=f[7]+(f[6]<<8|f[6]>>>24)+f[5]|0}a.Rabbit=s._createHelper(h)})(),n.Rabbit})})(hl)),hl.exports}var ml={exports:{}},DE=ml.exports,em;function TE(){return em||(em=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(DE,function(n){return(function(){var a=n,i=a.lib,s=i.StreamCipher,c=a.algo,u=[],p=[],f=[],h=c.RabbitLegacy=s.extend({_doReset:function(){var y=this._key.words,w=this.cfg.iv,E=this._X=[y[0],y[3]<<16|y[2]>>>16,y[1],y[0]<<16|y[3]>>>16,y[2],y[1]<<16|y[0]>>>16,y[3],y[2]<<16|y[1]>>>16],C=this._C=[y[2]<<16|y[2]>>>16,y[0]&4294901760|y[1]&65535,y[3]<<16|y[3]>>>16,y[1]&4294901760|y[2]&65535,y[0]<<16|y[0]>>>16,y[2]&4294901760|y[3]&65535,y[1]<<16|y[1]>>>16,y[3]&4294901760|y[0]&65535];this._b=0;for(var S=0;S<4;S++)x.call(this);for(var S=0;S<8;S++)C[S]^=E[S+4&7];if(w){var P=w.words,b=P[0],A=P[1],T=(b<<8|b>>>24)&16711935|(b<<24|b>>>8)&4278255360,B=(A<<8|A>>>24)&16711935|(A<<24|A>>>8)&4278255360,F=T>>>16|B&4294901760,k=B<<16|T&65535;C[0]^=T,C[1]^=F,C[2]^=B,C[3]^=k,C[4]^=T,C[5]^=F,C[6]^=B,C[7]^=k;for(var S=0;S<4;S++)x.call(this)}},_doProcessBlock:function(y,w){var E=this._X;x.call(this),u[0]=E[0]^E[5]>>>16^E[3]<<16,u[1]=E[2]^E[7]>>>16^E[5]<<16,u[2]=E[4]^E[1]>>>16^E[7]<<16,u[3]=E[6]^E[3]>>>16^E[1]<<16;for(var C=0;C<4;C++)u[C]=(u[C]<<8|u[C]>>>24)&16711935|(u[C]<<24|u[C]>>>8)&4278255360,y[w+C]^=u[C]},blockSize:128/32,ivSize:64/32});function x(){for(var y=this._X,w=this._C,E=0;E<8;E++)p[E]=w[E];w[0]=w[0]+1295307597+this._b|0,w[1]=w[1]+3545052371+(w[0]>>>0<p[0]>>>0?1:0)|0,w[2]=w[2]+886263092+(w[1]>>>0<p[1]>>>0?1:0)|0,w[3]=w[3]+1295307597+(w[2]>>>0<p[2]>>>0?1:0)|0,w[4]=w[4]+3545052371+(w[3]>>>0<p[3]>>>0?1:0)|0,w[5]=w[5]+886263092+(w[4]>>>0<p[4]>>>0?1:0)|0,w[6]=w[6]+1295307597+(w[5]>>>0<p[5]>>>0?1:0)|0,w[7]=w[7]+3545052371+(w[6]>>>0<p[6]>>>0?1:0)|0,this._b=w[7]>>>0<p[7]>>>0?1:0;for(var E=0;E<8;E++){var C=y[E]+w[E],S=C&65535,P=C>>>16,b=((S*S>>>17)+S*P>>>15)+P*P,A=((C&4294901760)*C|0)+((C&65535)*C|0);f[E]=b^A}y[0]=f[0]+(f[7]<<16|f[7]>>>16)+(f[6]<<16|f[6]>>>16)|0,y[1]=f[1]+(f[0]<<8|f[0]>>>24)+f[7]|0,y[2]=f[2]+(f[1]<<16|f[1]>>>16)+(f[0]<<16|f[0]>>>16)|0,y[3]=f[3]+(f[2]<<8|f[2]>>>24)+f[1]|0,y[4]=f[4]+(f[3]<<16|f[3]>>>16)+(f[2]<<16|f[2]>>>16)|0,y[5]=f[5]+(f[4]<<8|f[4]>>>24)+f[3]|0,y[6]=f[6]+(f[5]<<16|f[5]>>>16)+(f[4]<<16|f[4]>>>16)|0,y[7]=f[7]+(f[6]<<8|f[6]>>>24)+f[5]|0}a.RabbitLegacy=s._createHelper(h)})(),n.RabbitLegacy})})(ml)),ml.exports}var gl={exports:{}},_E=gl.exports,tm;function LE(){return tm||(tm=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ro(),Ao(),Jn(),$t())})(_E,function(n){return(function(){var a=n,i=a.lib,s=i.BlockCipher,c=a.algo;const u=16,p=[608135816,2242054355,320440878,57701188,2752067618,698298832,137296536,3964562569,1160258022,953160567,3193202383,887688300,3232508343,3380367581,1065670069,3041331479,2450970073,2306472731],f=[[3509652390,2564797868,805139163,3491422135,3101798381,1780907670,3128725573,4046225305,614570311,3012652279,134345442,2240740374,1667834072,1901547113,2757295779,4103290238,227898511,1921955416,1904987480,2182433518,2069144605,3260701109,2620446009,720527379,3318853667,677414384,3393288472,3101374703,2390351024,1614419982,1822297739,2954791486,3608508353,3174124327,2024746970,1432378464,3864339955,2857741204,1464375394,1676153920,1439316330,715854006,3033291828,289532110,2706671279,2087905683,3018724369,1668267050,732546397,1947742710,3462151702,2609353502,2950085171,1814351708,2050118529,680887927,999245976,1800124847,3300911131,1713906067,1641548236,4213287313,1216130144,1575780402,4018429277,3917837745,3693486850,3949271944,596196993,3549867205,258830323,2213823033,772490370,2760122372,1774776394,2652871518,566650946,4142492826,1728879713,2882767088,1783734482,3629395816,2517608232,2874225571,1861159788,326777828,3124490320,2130389656,2716951837,967770486,1724537150,2185432712,2364442137,1164943284,2105845187,998989502,3765401048,2244026483,1075463327,1455516326,1322494562,910128902,469688178,1117454909,936433444,3490320968,3675253459,1240580251,122909385,2157517691,634681816,4142456567,3825094682,3061402683,2540495037,79693498,3249098678,1084186820,1583128258,426386531,1761308591,1047286709,322548459,995290223,1845252383,2603652396,3431023940,2942221577,3202600964,3727903485,1712269319,422464435,3234572375,1170764815,3523960633,3117677531,1434042557,442511882,3600875718,1076654713,1738483198,4213154764,2393238008,3677496056,1014306527,4251020053,793779912,2902807211,842905082,4246964064,1395751752,1040244610,2656851899,3396308128,445077038,3742853595,3577915638,679411651,2892444358,2354009459,1767581616,3150600392,3791627101,3102740896,284835224,4246832056,1258075500,768725851,2589189241,3069724005,3532540348,1274779536,3789419226,2764799539,1660621633,3471099624,4011903706,913787905,3497959166,737222580,2514213453,2928710040,3937242737,1804850592,3499020752,2949064160,2386320175,2390070455,2415321851,4061277028,2290661394,2416832540,1336762016,1754252060,3520065937,3014181293,791618072,3188594551,3933548030,2332172193,3852520463,3043980520,413987798,3465142937,3030929376,4245938359,2093235073,3534596313,375366246,2157278981,2479649556,555357303,3870105701,2008414854,3344188149,4221384143,3956125452,2067696032,3594591187,2921233993,2428461,544322398,577241275,1471733935,610547355,4027169054,1432588573,1507829418,2025931657,3646575487,545086370,48609733,2200306550,1653985193,298326376,1316178497,3007786442,2064951626,458293330,2589141269,3591329599,3164325604,727753846,2179363840,146436021,1461446943,4069977195,705550613,3059967265,3887724982,4281599278,3313849956,1404054877,2845806497,146425753,1854211946],[1266315497,3048417604,3681880366,3289982499,290971e4,1235738493,2632868024,2414719590,3970600049,1771706367,1449415276,3266420449,422970021,1963543593,2690192192,3826793022,1062508698,1531092325,1804592342,2583117782,2714934279,4024971509,1294809318,4028980673,1289560198,2221992742,1669523910,35572830,157838143,1052438473,1016535060,1802137761,1753167236,1386275462,3080475397,2857371447,1040679964,2145300060,2390574316,1461121720,2956646967,4031777805,4028374788,33600511,2920084762,1018524850,629373528,3691585981,3515945977,2091462646,2486323059,586499841,988145025,935516892,3367335476,2599673255,2839830854,265290510,3972581182,2759138881,3795373465,1005194799,847297441,406762289,1314163512,1332590856,1866599683,4127851711,750260880,613907577,1450815602,3165620655,3734664991,3650291728,3012275730,3704569646,1427272223,778793252,1343938022,2676280711,2052605720,1946737175,3164576444,3914038668,3967478842,3682934266,1661551462,3294938066,4011595847,840292616,3712170807,616741398,312560963,711312465,1351876610,322626781,1910503582,271666773,2175563734,1594956187,70604529,3617834859,1007753275,1495573769,4069517037,2549218298,2663038764,504708206,2263041392,3941167025,2249088522,1514023603,1998579484,1312622330,694541497,2582060303,2151582166,1382467621,776784248,2618340202,3323268794,2497899128,2784771155,503983604,4076293799,907881277,423175695,432175456,1378068232,4145222326,3954048622,3938656102,3820766613,2793130115,2977904593,26017576,3274890735,3194772133,1700274565,1756076034,4006520079,3677328699,720338349,1533947780,354530856,688349552,3973924725,1637815568,332179504,3949051286,53804574,2852348879,3044236432,1282449977,3583942155,3416972820,4006381244,1617046695,2628476075,3002303598,1686838959,431878346,2686675385,1700445008,1080580658,1009431731,832498133,3223435511,2605976345,2271191193,2516031870,1648197032,4164389018,2548247927,300782431,375919233,238389289,3353747414,2531188641,2019080857,1475708069,455242339,2609103871,448939670,3451063019,1395535956,2413381860,1841049896,1491858159,885456874,4264095073,4001119347,1565136089,3898914787,1108368660,540939232,1173283510,2745871338,3681308437,4207628240,3343053890,4016749493,1699691293,1103962373,3625875870,2256883143,3830138730,1031889488,3479347698,1535977030,4236805024,3251091107,2132092099,1774941330,1199868427,1452454533,157007616,2904115357,342012276,595725824,1480756522,206960106,497939518,591360097,863170706,2375253569,3596610801,1814182875,2094937945,3421402208,1082520231,3463918190,2785509508,435703966,3908032597,1641649973,2842273706,3305899714,1510255612,2148256476,2655287854,3276092548,4258621189,236887753,3681803219,274041037,1734335097,3815195456,3317970021,1899903192,1026095262,4050517792,356393447,2410691914,3873677099,3682840055],[3913112168,2491498743,4132185628,2489919796,1091903735,1979897079,3170134830,3567386728,3557303409,857797738,1136121015,1342202287,507115054,2535736646,337727348,3213592640,1301675037,2528481711,1895095763,1721773893,3216771564,62756741,2142006736,835421444,2531993523,1442658625,3659876326,2882144922,676362277,1392781812,170690266,3921047035,1759253602,3611846912,1745797284,664899054,1329594018,3901205900,3045908486,2062866102,2865634940,3543621612,3464012697,1080764994,553557557,3656615353,3996768171,991055499,499776247,1265440854,648242737,3940784050,980351604,3713745714,1749149687,3396870395,4211799374,3640570775,1161844396,3125318951,1431517754,545492359,4268468663,3499529547,1437099964,2702547544,3433638243,2581715763,2787789398,1060185593,1593081372,2418618748,4260947970,69676912,2159744348,86519011,2512459080,3838209314,1220612927,3339683548,133810670,1090789135,1078426020,1569222167,845107691,3583754449,4072456591,1091646820,628848692,1613405280,3757631651,526609435,236106946,48312990,2942717905,3402727701,1797494240,859738849,992217954,4005476642,2243076622,3870952857,3732016268,765654824,3490871365,2511836413,1685915746,3888969200,1414112111,2273134842,3281911079,4080962846,172450625,2569994100,980381355,4109958455,2819808352,2716589560,2568741196,3681446669,3329971472,1835478071,660984891,3704678404,4045999559,3422617507,3040415634,1762651403,1719377915,3470491036,2693910283,3642056355,3138596744,1364962596,2073328063,1983633131,926494387,3423689081,2150032023,4096667949,1749200295,3328846651,309677260,2016342300,1779581495,3079819751,111262694,1274766160,443224088,298511866,1025883608,3806446537,1145181785,168956806,3641502830,3584813610,1689216846,3666258015,3200248200,1692713982,2646376535,4042768518,1618508792,1610833997,3523052358,4130873264,2001055236,3610705100,2202168115,4028541809,2961195399,1006657119,2006996926,3186142756,1430667929,3210227297,1314452623,4074634658,4101304120,2273951170,1399257539,3367210612,3027628629,1190975929,2062231137,2333990788,2221543033,2438960610,1181637006,548689776,2362791313,3372408396,3104550113,3145860560,296247880,1970579870,3078560182,3769228297,1714227617,3291629107,3898220290,166772364,1251581989,493813264,448347421,195405023,2709975567,677966185,3703036547,1463355134,2715995803,1338867538,1343315457,2802222074,2684532164,233230375,2599980071,2000651841,3277868038,1638401717,4028070440,3237316320,6314154,819756386,300326615,590932579,1405279636,3267499572,3150704214,2428286686,3959192993,3461946742,1862657033,1266418056,963775037,2089974820,2263052895,1917689273,448879540,3550394620,3981727096,150775221,3627908307,1303187396,508620638,2975983352,2726630617,1817252668,1876281319,1457606340,908771278,3720792119,3617206836,2455994898,1729034894,1080033504],[976866871,3556439503,2881648439,1522871579,1555064734,1336096578,3548522304,2579274686,3574697629,3205460757,3593280638,3338716283,3079412587,564236357,2993598910,1781952180,1464380207,3163844217,3332601554,1699332808,1393555694,1183702653,3581086237,1288719814,691649499,2847557200,2895455976,3193889540,2717570544,1781354906,1676643554,2592534050,3230253752,1126444790,2770207658,2633158820,2210423226,2615765581,2414155088,3127139286,673620729,2805611233,1269405062,4015350505,3341807571,4149409754,1057255273,2012875353,2162469141,2276492801,2601117357,993977747,3918593370,2654263191,753973209,36408145,2530585658,25011837,3520020182,2088578344,530523599,2918365339,1524020338,1518925132,3760827505,3759777254,1202760957,3985898139,3906192525,674977740,4174734889,2031300136,2019492241,3983892565,4153806404,3822280332,352677332,2297720250,60907813,90501309,3286998549,1016092578,2535922412,2839152426,457141659,509813237,4120667899,652014361,1966332200,2975202805,55981186,2327461051,676427537,3255491064,2882294119,3433927263,1307055953,942726286,933058658,2468411793,3933900994,4215176142,1361170020,2001714738,2830558078,3274259782,1222529897,1679025792,2729314320,3714953764,1770335741,151462246,3013232138,1682292957,1483529935,471910574,1539241949,458788160,3436315007,1807016891,3718408830,978976581,1043663428,3165965781,1927990952,4200891579,2372276910,3208408903,3533431907,1412390302,2931980059,4132332400,1947078029,3881505623,4168226417,2941484381,1077988104,1320477388,886195818,18198404,3786409e3,2509781533,112762804,3463356488,1866414978,891333506,18488651,661792760,1628790961,3885187036,3141171499,876946877,2693282273,1372485963,791857591,2686433993,3759982718,3167212022,3472953795,2716379847,445679433,3561995674,3504004811,3574258232,54117162,3331405415,2381918588,3769707343,4154350007,1140177722,4074052095,668550556,3214352940,367459370,261225585,2610173221,4209349473,3468074219,3265815641,314222801,3066103646,3808782860,282218597,3406013506,3773591054,379116347,1285071038,846784868,2669647154,3771962079,3550491691,2305946142,453669953,1268987020,3317592352,3279303384,3744833421,2610507566,3859509063,266596637,3847019092,517658769,3462560207,3443424879,370717030,4247526661,2224018117,4143653529,4112773975,2788324899,2477274417,1456262402,2901442914,1517677493,1846949527,2295493580,3734397586,2176403920,1280348187,1908823572,3871786941,846861322,1172426758,3287448474,3383383037,1655181056,3139813346,901632758,1897031941,2986607138,3066810236,3447102507,1393639104,373351379,950779232,625454576,3124240540,4148612726,2007998917,544563296,2244738638,2330496472,2058025392,1291430526,424198748,50039436,29584100,3605783033,2429876329,2791104160,1057563949,3255363231,3075367218,3463963227,1469046755,985887462]];var h={pbox:[],sbox:[]};function x(S,P){let b=P>>24&255,A=P>>16&255,T=P>>8&255,B=P&255,F=S.sbox[0][b]+S.sbox[1][A];return F=F^S.sbox[2][T],F=F+S.sbox[3][B],F}function y(S,P,b){let A=P,T=b,B;for(let F=0;F<u;++F)A=A^S.pbox[F],T=x(S,A)^T,B=A,A=T,T=B;return B=A,A=T,T=B,T=T^S.pbox[u],A=A^S.pbox[u+1],{left:A,right:T}}function w(S,P,b){let A=P,T=b,B;for(let F=u+1;F>1;--F)A=A^S.pbox[F],T=x(S,A)^T,B=A,A=T,T=B;return B=A,A=T,T=B,T=T^S.pbox[1],A=A^S.pbox[0],{left:A,right:T}}function E(S,P,b){for(let k=0;k<4;k++){S.sbox[k]=[];for(let N=0;N<256;N++)S.sbox[k][N]=f[k][N]}let A=0;for(let k=0;k<u+2;k++)S.pbox[k]=p[k]^P[A],A++,A>=b&&(A=0);let T=0,B=0,F=0;for(let k=0;k<u+2;k+=2)F=y(S,T,B),T=F.left,B=F.right,S.pbox[k]=T,S.pbox[k+1]=B;for(let k=0;k<4;k++)for(let N=0;N<256;N+=2)F=y(S,T,B),T=F.left,B=F.right,S.sbox[k][N]=T,S.sbox[k][N+1]=B;return!0}var C=c.Blowfish=s.extend({_doReset:function(){if(this._keyPriorReset!==this._key){var S=this._keyPriorReset=this._key,P=S.words,b=S.sigBytes/4;E(h,P,b)}},encryptBlock:function(S,P){var b=y(h,S[P],S[P+1]);S[P]=b.left,S[P+1]=b.right},decryptBlock:function(S,P){var b=w(h,S[P],S[P+1]);S[P]=b.left,S[P+1]=b.right},blockSize:64/32,keySize:128/32,ivSize:64/32});a.Blowfish=s._createHelper(C)})(),n.Blowfish})})(gl)),gl.exports}var FE=Is.exports,rm;function NE(){return rm||(rm=1,(function(r,t){(function(n,a,i){r.exports=a(Je(),Ml(),T2(),L2(),Ro(),I2(),Ao(),yg(),Ld(),$2(),wg(),H2(),W2(),q2(),Fd(),Y2(),Jn(),$t(),eE(),rE(),oE(),iE(),lE(),uE(),fE(),hE(),gE(),vE(),wE(),CE(),SE(),AE(),kE(),TE(),LE())})(FE,function(n){return n})})(Is)),Is.exports}var IE=NE();const kt=Tm(IE),BE=1e5,jE=256/32,OE=16,Eg="neon-motion-platform-v1";let Gu=null,nm=null;function $l(r){return r?UE(r):crypto.getRandomValues(new Uint8Array(OE))}function Cg(r){return bg(r)}function Nd(r,t){const n=kt.lib.WordArray.create(t),a=r+":"+bg(t);if(Gu&&nm===a)return kt.enc.Hex.parse(Gu);const i=kt.PBKDF2(r,n,{keySize:jE,iterations:BE});return Gu=i.toString(),nm=a,i}async function oi(r,t){const n=Nd(Eg,t),a=kt.lib.WordArray.random(128/8);return{ciphertext:kt.AES.encrypt(r,n,{iv:a,mode:kt.mode.CBC,padding:kt.pad.Pkcs7}).ciphertext.toString(kt.enc.Base64),iv:a.toString(kt.enc.Base64)}}async function Id(r,t){const n=Nd(Eg,t),a=kt.enc.Base64.parse(r.iv),i=kt.enc.Base64.parse(r.ciphertext),c=kt.AES.decrypt({ciphertext:i},n,{iv:a,mode:kt.mode.CBC,padding:kt.pad.Pkcs7}).toString(kt.enc.Utf8);if(!c)throw new Error("解密失败");return c}async function ME(r,t){try{const n=$E(),a=Nd(n,t),i=kt.enc.Base64.parse(r.iv),s=kt.enc.Base64.parse(r.ciphertext);return kt.AES.decrypt({ciphertext:s},a,{iv:i,mode:kt.mode.CBC,padding:kt.pad.Pkcs7}).toString(kt.enc.Utf8)||null}catch{return null}}function $E(){return[navigator.userAgent||"",navigator.language||"",navigator.platform||"",String(navigator.hardwareConcurrency||0),Intl.DateTimeFormat().resolvedOptions().timeZone||""].join("|")}function bg(r){let t="";for(let n=0;n<r.length;n++)t+=String.fromCharCode(r[n]);return btoa(t)}function UE(r){const t=atob(r),n=new Uint8Array(t.length);for(let a=0;a<t.length;a++)n[a]=t.charCodeAt(a);return n}function pi(){if(typeof crypto<"u"&&typeof crypto.randomUUID=="function")try{return crypto.randomUUID()}catch{}if(typeof crypto<"u"&&typeof crypto.getRandomValues=="function")try{const r=new Uint8Array(16);crypto.getRandomValues(r),r[6]=r[6]&15|64,r[8]=r[8]&63|128;const t=Array.from(r,n=>(n<16?"0":"")+n.toString(16)).join("");return[t.slice(0,8),t.slice(8,12),t.slice(12,16),t.slice(16,20),t.slice(20,32)].join("-")}catch{}return"xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g,r=>{const t=Math.random()*16|0;return(r==="x"?t:t&3|8).toString(16)})}const st={LLM_CONFIG:"motion-platform:llm-config",CHAT_HISTORY:"motion-platform:chat-history",LAST_MOTION:"motion-platform:last-motion",PROMPT_OPTIMIZATION_ENABLED:"motion-platform:prompt-optimization-enabled",ASPECT_RATIO:"motion-platform:aspect-ratio",CLARIFY_SESSION:"motion-platform:clarify-session",CLARIFY_ENABLED:"motion-platform:clarify-enabled",CONVERSATION_INDEX:"motion-platform:conversation-index",CURRENT_CONVERSATION:"motion-platform:current-conversation",CONVERSATION_PREFIX:"motion-platform:conversation:",LLM_CONFIGS:"motion-platform:llm-configs",CRYPTO_SALT:"motion-platform:crypto-salt"};function Eo(r,t){if(!r)return t;try{return JSON.parse(r)}catch{return t}}function Br(r){try{return localStorage.getItem(r)}catch{return console.warn(`Failed to read from localStorage: ${r}`),null}}function jr(r,t){try{localStorage.setItem(r,t)}catch(n){n instanceof DOMException&&(n.name==="QuotaExceededError"||n.code===22)?console.error(`[Storage] 存储空间不足，无法保存: ${r}。请考虑删除一些历史对话。`):console.warn(`Failed to write to localStorage: ${r}`,n)}}function Co(r){try{localStorage.removeItem(r)}catch{console.warn(`Failed to remove from localStorage: ${r}`)}}const er={saveLLMConfig(r){jr(st.LLM_CONFIG,JSON.stringify(r))},getLLMConfig(){const r=Br(st.LLM_CONFIG);return Eo(r,null)},saveChatHistory(r){jr(st.CHAT_HISTORY,JSON.stringify(r))},getChatHistory(){const r=Br(st.CHAT_HISTORY);return Eo(r,[])},saveLastMotion(r){jr(st.LAST_MOTION,JSON.stringify(r))},getLastMotion(){const r=Br(st.LAST_MOTION);return Eo(r,null)},clearAll(){Co(st.LLM_CONFIG),Co(st.CHAT_HISTORY),Co(st.LAST_MOTION)}};function zE(){const r=Br(st.ASPECT_RATIO);return r&&["16:9","4:3","1:1","9:16","21:9","2.35:1"].includes(r)?r:null}function HE(r){jr(st.ASPECT_RATIO,r)}function ws(r){jr(st.CLARIFY_SESSION,JSON.stringify(r))}function VE(){const r=Br(st.CLARIFY_SESSION);return Eo(r,null)}function WE(){Co(st.CLARIFY_SESSION)}function GE(){const r=Br(st.CLARIFY_ENABLED);return r===null?!0:r==="true"}function qE(r){jr(st.CLARIFY_ENABLED,String(r))}function KE(){const r=Br(st.CONVERSATION_INDEX);return Eo(r,null)}function ln(r){jr(st.CONVERSATION_INDEX,JSON.stringify(r))}function XE(){return Br(st.CURRENT_CONVERSATION)}function cn(r){r===null?Co(st.CURRENT_CONVERSATION):jr(st.CURRENT_CONVERSATION,r)}function YE(r){if(!r||typeof r!="object")return!1;const t=r;return typeof t.id=="string"&&typeof t.title=="string"&&Array.isArray(t.messages)&&typeof t.createdAt=="number"&&typeof t.updatedAt=="number"}function dn(r){const t=`${st.CONVERSATION_PREFIX}${r}`,n=Br(t),a=Eo(n,null);return a&&!YE(a)?(console.warn(`[Storage] 对话数据损坏: ${r}，将被忽略`),null):a}function vo(r){const t=`${st.CONVERSATION_PREFIX}${r.id}`;jr(t,JSON.stringify(r))}function QE(r){const t=`${st.CONVERSATION_PREFIX}${r}`;Co(t)}function Ul(){return Br(st.CRYPTO_SALT)}function Sg(r){jr(st.CRYPTO_SALT,r)}function Rg(){const r=Br(st.LLM_CONFIGS);return Eo(r,null)}function Vn(r){jr(st.LLM_CONFIGS,JSON.stringify(r))}function JE(r){try{const n=new URL(r).hostname;return n.includes("openai")?"OpenAI":n.includes("deepseek")?"DeepSeek":n.includes("anthropic")?"Anthropic":n.includes("azure")?"Azure":n.split(".")[0]}catch{return"Unknown"}}async function ZE(){if(Rg())return null;const t=er.getLLMConfig();if(!t)return null;console.log("[Storage] 检测到旧版配置，开始迁移到多配置格式...");try{const n=Ul(),a=$l(n);n||Sg(Cg(a));const i=await oi(t.apiKey,a),c=`${JE(t.baseURL)} - ${t.model}`,u=new Date().toISOString(),p={id:pi(),name:c,baseURL:t.baseURL,apiKey:i,model:t.model,createdAt:u,updatedAt:u},f={configs:[p],activeConfigId:p.id};return Vn(f),Co(st.LLM_CONFIG),console.log("[Storage] 配置迁移完成"),f}catch(n){return console.error("[Storage] 配置迁移失败:",n),null}}const xl=new Map;async function eC(r){if(!na.ACCEPTED_FORMATS.includes(r.type))return!1;const t=await r.slice(0,8).arrayBuffer(),n=new Uint8Array(t);return!!(Rl.PNG.every((s,c)=>n[c]===s)&&r.type==="image/png"||Rl.JPEG.every((s,c)=>n[c]===s)&&r.type==="image/jpeg")}function tC(r){return r.size<=na.MAX_FILE_SIZE}function Ag(r){return new Promise((t,n)=>{const a=URL.createObjectURL(r),i=new Image;i.onload=()=>{URL.revokeObjectURL(a),t({width:i.naturalWidth,height:i.naturalHeight})},i.onerror=()=>{URL.revokeObjectURL(a),n(new Error("无法读取图片尺寸"))},i.src=a})}async function rC(r,t=na.MAX_DIMENSION){const n=URL.createObjectURL(r);try{const a=await zl(n),{width:i,height:s}=a;let c=i,u=s;if(i>t||s>t){const h=Math.min(t/i,t/s);c=Math.round(i*h),u=Math.round(s*h)}else return r;const p=document.createElement("canvas");p.width=c,p.height=u;const f=p.getContext("2d");return f.imageSmoothingEnabled=!0,f.imageSmoothingQuality="high",f.drawImage(a,0,0,c,u),new Promise((h,x)=>{p.toBlob(y=>{y?h(y):x(new Error("图片缩放失败"))},r.type,.92)})}finally{URL.revokeObjectURL(n)}}function zl(r){return new Promise((t,n)=>{const a=new Image;a.onload=()=>t(a),a.onerror=()=>n(new Error("图片加载失败")),a.src=r})}async function nC(r){if(!await eC(r))throw new Error("INVALID_FORMAT");if(!tC(r))throw new Error("FILE_TOO_LARGE");let n;try{n=await Ag(r)}catch{throw new Error("CORRUPTED_FILE")}let a=r,i=!1;if(n.width>na.MAX_DIMENSION||n.height>na.MAX_DIMENSION){a=await rC(r),i=!0;const c=URL.createObjectURL(a);try{const u=await zl(c);n={width:u.naturalWidth,height:u.naturalHeight}}finally{URL.revokeObjectURL(c)}}return{blobUrl:URL.createObjectURL(a),originalFileName:r.name,width:n.width,height:n.height,wasResized:i}}async function vl(r){if(xl.has(r))return xl.get(r);if(r===Td){const n=Al();return vl(n)}const t=await zl(r);return xl.set(r,t),t}function Al(r=200,t=200){const n=`
    <svg xmlns="http://www.w3.org/2000/svg" width="${r}" height="${t}" viewBox="0 0 ${r} ${t}">
      <rect width="100%" height="100%" fill="#e5e7eb"/>
      <g fill="#9ca3af" transform="translate(${r/2-24}, ${t/2-24})">
        <path d="M4 4h40v40H4z" fill="none" stroke="#9ca3af" stroke-width="2"/>
        <circle cx="16" cy="16" r="4"/>
        <path d="M4 36l10-12 8 8 12-16 10 20"/>
      </g>
    </svg>
  `.trim();return`data:image/svg+xml,${encodeURIComponent(n)}`}function Es(r){r&&r.startsWith("blob:")&&(URL.revokeObjectURL(r),xl.delete(r))}function Pg(r){return r===Td||!r}const oC=[82,73,70,70],aC=[87,69,66,80];async function iC(r){if(!Ir.ACCEPTED_IMAGE_FORMATS.includes(r.type))return!1;const n=await r.slice(0,12).arrayBuffer(),a=new Uint8Array(n);if(Rl.PNG.every((p,f)=>a[f]===p)&&r.type==="image/png"||Rl.JPEG.every((p,f)=>a[f]===p)&&r.type==="image/jpeg")return!0;const c=oC.every((p,f)=>a[f]===p),u=aC.every((p,f)=>a[f+8]===p);return!!(c&&u&&r.type==="image/webp")}async function sC(r,t=Ir.IMAGE_SHORT_EDGE_LIMIT){const n=URL.createObjectURL(r);try{const a=await zl(n),{width:i,height:s}={width:a.naturalWidth,height:a.naturalHeight},c=Math.min(i,s);if(c<=t)return{blob:r,wasCompressed:!1,dimensions:{width:i,height:s}};const u=t/c,p=Math.round(i*u),f=Math.round(s*u),h=document.createElement("canvas");h.width=p,h.height=f;const x=h.getContext("2d");x.imageSmoothingEnabled=!0,x.imageSmoothingQuality="high",x.drawImage(a,0,0,p,f);const y=r.type==="image/png"?"image/png":"image/jpeg",w=.92;return new Promise((E,C)=>{h.toBlob(S=>{S?E({blob:S,wasCompressed:!0,dimensions:{width:p,height:f}}):C(new Error("图片压缩失败"))},y,w)})}finally{URL.revokeObjectURL(n)}}async function lC(r){return new Promise((t,n)=>{const a=new FileReader;a.onload=()=>t(a.result),a.onerror=()=>n(new Error("图片转换失败")),a.readAsDataURL(r)})}const Ka=new Map;function cC(r){return Ol.ACCEPTED_FORMATS.includes(r.type)}function uC(r){return r.size<=Ol.MAX_FILE_SIZE}function dC(r){return new Promise((t,n)=>{const a=URL.createObjectURL(r),i=document.createElement("video");i.preload="metadata",i.onloadedmetadata=()=>{URL.revokeObjectURL(a),t({duration:i.duration*1e3,width:i.videoWidth,height:i.videoHeight})},i.onerror=()=>{URL.revokeObjectURL(a),n(new Error("CORRUPTED_FILE"))},i.src=a})}function fC(r){return r<=Ol.MAX_DURATION}async function pC(r){if(!cC(r))throw new Error("INVALID_FORMAT");if(!uC(r))throw new Error("FILE_TOO_LARGE");let t;try{t=await dC(r)}catch{throw new Error("CORRUPTED_FILE")}if(!fC(t.duration))throw new Error("DURATION_TOO_LONG");return{blobUrl:URL.createObjectURL(r),originalFileName:r.name,duration:t.duration,width:t.width,height:t.height,wasResized:!1}}function om(r){return new Promise((t,n)=>{const a=document.createElement("video");a.muted=!0,a.playsInline=!0,a.preload="auto",a.oncanplaythrough=()=>{t(a)},a.onerror=()=>n(new Error("LOAD_ERROR")),a.src=r,a.load()})}async function qu(r){if(Ka.has(r))return Ka.get(r);if(r===_d||!r){const n=Pl(),a=await om(n);return Ka.set(r,a),a}const t=await om(r);return Ka.set(r,t),t}function Pl(r=200,t=200){const n=`
    <svg xmlns="http://www.w3.org/2000/svg" width="${r}" height="${t}" viewBox="0 0 ${r} ${t}">
      <rect width="100%" height="100%" fill="#1f2937"/>
      <g fill="#6b7280" transform="translate(${r/2-24}, ${t/2-24})">
        <rect x="4" y="4" width="40" height="40" rx="4" fill="none" stroke="#6b7280" stroke-width="2"/>
        <polygon points="18,14 18,34 34,24" fill="#6b7280"/>
      </g>
      <text x="${r/2}" y="${t-20}" text-anchor="middle" fill="#6b7280" font-size="12" font-family="sans-serif">
        视频占位符
      </text>
    </svg>
  `.trim();return`data:image/svg+xml,${encodeURIComponent(n)}`}function am(r){r&&r.startsWith("blob:")&&(URL.revokeObjectURL(r),Ka.delete(r))}function kg(r){return r===_d||!r}const Dg=[{id:"16:9",label:"aspectRatio.16_9",ratio:16/9,isVertical:!1},{id:"4:3",label:"aspectRatio.4_3",ratio:4/3,isVertical:!1},{id:"1:1",label:"aspectRatio.1_1",ratio:1,isVertical:!1},{id:"9:16",label:"aspectRatio.9_16",ratio:9/16,isVertical:!0},{id:"21:9",label:"aspectRatio.21_9",ratio:21/9,isVertical:!1},{id:"2.35:1",label:"aspectRatio.2_35_1",ratio:2.35,isVertical:!1}],hC=[{id:"720p",label:"720p (HD)",baseHeight:720},{id:"1080p",label:"1080p (Full HD)",baseHeight:1080},{id:"4k",label:"4K (Ultra HD)",baseHeight:2160}],Hl="16:9";function Vl(r){const t=Dg.find(n=>n.id===r);if(!t)throw new Error(`无效的画面比例: ${r}`);return t}function mC(r){const t=hC.find(n=>n.id===r);if(!t)throw new Error(`无效的导出分辨率: ${r}`);return t}function Wl(r,t){const n=Vl(r),i=mC(t).baseHeight;if(n.ratio>=1){const s=i;return{width:Math.round(s*n.ratio),height:s}}else{const s=i,c=Math.round(s/n.ratio);return{width:s,height:c}}}function gC(r,t,n){const a=Vl(r),i=t/n,s=a.ratio;let c,u;return i>s?(u=n,c=Math.round(n*s)):(c=t,u=Math.round(t/s)),{width:c,height:u}}function xC(){const r=localStorage.getItem("neon-locale");return r==="zh"||r==="en"?r:navigator.language.startsWith("en")?"en":"zh"}async function im(r){const t=$l(Ul());let n=!1;const a=[];for(const i of r){try{await Id(i.apiKey,t),a.push(i);continue}catch{}const s=await ME(i.apiKey,t);if(s){const c=await oi(s,t);a.push({...i,apiKey:c,updatedAt:new Date().toISOString()}),n=!0,console.log(`[AppStore] 已迁移配置 "${i.name}" 到新加密方式`)}else{const c=await oi("",t);a.push({...i,apiKey:c,updatedAt:new Date().toISOString()}),n=!0,console.warn(`[AppStore] 配置 "${i.name}" 无法迁移，已清空 API Key，请重新填写`)}}return{configs:a,changed:n}}const vC={currentMotion:null,messages:[],llmConfig:null,isGenerating:!1,isExporting:!1,exportProgress:0,error:null,isPlaying:!1,currentTime:0,isSettingsOpen:!1,isExportDialogOpen:!1,aspectRatio:Hl,clarifySession:null,isClarifying:!1,clarifyEnabled:!0,renderError:null,isFixing:!1,fixAttemptCount:0,fixAbortController:null,conversationList:[],currentConversationId:null,isHistoryPanelOpen:!1,llmConfigs:[],activeConfigId:null,isLoadingConfigs:!1,isAssetPackExportDialogOpen:!1,assetPackExportState:Ns,pendingAttachments:[],previewBackgroundUrl:null,toasts:[],currentPath:"/",isMobileMenuOpen:!1,locale:xC()},Cs=pi;function Ku(r){const t=r.find(a=>a.role==="user");if(!t)return Ct.t(gg);const n=t.content.trim();return n.length<=yh?n:n.substring(0,yh)+"..."}function pd(){return Ct.t("copy.suffix")}function yC(){const r=hd(pd().replace(/[()（）]/g,""));return new RegExp(`^(.+?)\\s*[（(]${r}(?:\\s*(\\d+))?[）)]$`)}const sm=50;function hd(r){return r.replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}function wC(r,t){const n=yC(),a=r.match(n),i=a?a[1].trim():r.trim();let s=0;const c=pd().replace(/[()（）]/g,""),u=new RegExp(`^${hd(i)}\\s*[（(]${hd(c)}(?:\\s*(\\d+))?[）)]$`);for(const h of t){const x=h.match(u);if(x){const y=x[1]?parseInt(x[1],10):1;y>s&&(s=y)}}const p=s===0?pd():Ct.t("copy.suffixN",{n:s+1}),f=`${i} ${p}`;if(f.length>sm){const h=sm-p.length-1;return`${i.substring(0,h)} ${p}`}return f}const EC=300;let bs=null;const Ht=mg((r,t)=>({...vC,setCurrentMotion:n=>{const{currentMotion:a}=t();a&&a.parameters.forEach(i=>{i.type==="image"&&i.imageValue&&Es(i.imageValue),i.type==="video"&&i.videoValue&&am(i.videoValue)}),r({currentMotion:n,isPlaying:!1,currentTime:0}),n&&er.saveLastMotion(n)},updateMotionParameter:(n,a)=>{const{currentMotion:i}=t();if(!i)return;console.log("[AppStore] updateMotionParameter:",n,a);const s=i.parameters.map(u=>{if(u.id!==n)return u;switch(u.type){case"number":return{...u,value:a};case"color":return{...u,colorValue:a};case"select":return{...u,selectedValue:a};case"boolean":return{...u,boolValue:a};case"image":return u.imageValue&&Es(u.imageValue),{...u,imageValue:a};case"string":return{...u,stringValue:a};default:return u}}),c={...i,parameters:s,updatedAt:Date.now()};r({currentMotion:c}),er.saveLastMotion(c)},updateVideoParameter:(n,a,i)=>{const{currentMotion:s}=t();if(!s)return;console.log("[AppStore] updateVideoParameter:",n,i);const c=s.parameters.map(p=>p.id!==n?p:(p.videoValue&&am(p.videoValue),{...p,videoValue:a,videoFileName:i==null?void 0:i.originalFileName,videoDuration:i==null?void 0:i.duration,videoWidth:i==null?void 0:i.width,videoHeight:i==null?void 0:i.height})),u={...s,parameters:c,updatedAt:Date.now()};r({currentMotion:u}),er.saveLastMotion(u)},addMessage:n=>{const a=[...t().messages,n];r({messages:a}),er.saveChatHistory(a)},clearMessages:()=>{r({messages:[]}),er.saveChatHistory([])},setLLMConfig:n=>{r({llmConfig:n}),n&&er.saveLLMConfig(n)},setIsGenerating:n=>r({isGenerating:n}),setIsExporting:n=>r({isExporting:n}),setExportProgress:n=>r({exportProgress:n}),setError:n=>r({error:n}),setIsPlaying:n=>r({isPlaying:n}),setCurrentTime:n=>r({currentTime:n}),openSettings:()=>r({isSettingsOpen:!0}),closeSettings:()=>r({isSettingsOpen:!1}),openExportDialog:()=>r({isExportDialogOpen:!0,isPlaying:!1}),closeExportDialog:()=>r({isExportDialogOpen:!1}),setAspectRatio:n=>{r({aspectRatio:n}),HE(n)},setClarifySession:n=>{r({clarifySession:n}),n?ws(n):WE()},setIsClarifying:n=>r({isClarifying:n}),answerClarifyQuestion:(n,a)=>{const{clarifySession:i}=t();if(!i)return;const s=[...i.answers,a],c={...i,answers:s,updatedAt:Date.now()};r({clarifySession:c}),ws(c)},skipClarify:()=>{const{clarifySession:n}=t();if(!n)return;const a={...n,status:"skipped",updatedAt:Date.now()};r({clarifySession:a,isClarifying:!1}),ws(a)},nextClarifyQuestion:()=>{const{clarifySession:n}=t();if(!n)return;const a=n.currentQuestionIndex+1,i=a>=n.questions.length,s={...n,currentQuestionIndex:a,status:i?"completed":"questioning",updatedAt:Date.now()};r({clarifySession:s}),ws(s)},setClarifyEnabled:n=>{r({clarifyEnabled:n}),qE(n)},setRenderError:n=>{const{fixAbortController:a}=t();a&&a.abort(),r({renderError:n,fixAttemptCount:0,fixAbortController:null})},setIsFixing:n=>r({isFixing:n}),incrementFixAttempt:()=>{const{fixAttemptCount:n}=t();r({fixAttemptCount:n+1})},resetFixAttempt:()=>r({fixAttemptCount:0}),setFixAbortController:n=>r({fixAbortController:n}),clearErrorState:()=>{const{fixAbortController:n}=t();n&&n.abort(),r({renderError:null,isFixing:!1,fixAttemptCount:0,fixAbortController:null})},loadFromStorage:()=>{const n=er.getLLMConfig(),a=er.getChatHistory(),i=er.getLastMotion(),s=zE(),c=VE(),u=GE(),p=c,f=(c==null?void 0:c.status)==="questioning"||!1;r({llmConfig:n,messages:a,currentMotion:i,aspectRatio:s||Hl,clarifySession:p,isClarifying:f,clarifyEnabled:u})},saveToStorage:()=>{const{llmConfig:n,messages:a,currentMotion:i}=t();n&&er.saveLLMConfig(n),er.saveChatHistory(a),i&&er.saveLastMotion(i)},initConversations:()=>{let n=KE(),a=XE();if(!n){const i=er.getChatHistory(),s=er.getLastMotion();if(i.length>0||s){const c=Cs(),u=Date.now(),p=Ku(i);vo({id:c,title:p,messages:i,motion:s,createdAt:u,updatedAt:u}),n=[{id:c,title:p,updatedAt:u}],ln(n),a=c,cn(c)}else n=[]}if(n.length>0&&!a&&(a=[...n].sort((s,c)=>c.updatedAt-s.updatedAt)[0].id,cn(a)),a){const i=dn(a);if(i){r({conversationList:n,currentConversationId:a,messages:i.messages,currentMotion:i.motion});return}}r({conversationList:n,currentConversationId:null,messages:[],currentMotion:null})},createConversation:()=>{const{isGenerating:n,isClarifying:a,isFixing:i}=t();if(n||a||i)return;t().saveCurrentConversation(!0),t().clearPendingAttachments();const s=Cs(),c=Date.now(),u={id:s,title:Ct.t(gg),messages:[],motion:null,createdAt:c,updatedAt:c};vo(u);const{conversationList:p}=t(),h=[{id:s,title:u.title,updatedAt:c},...p];ln(h),cn(s),r({conversationList:h,currentConversationId:s,messages:[],currentMotion:null,clarifySession:null,isClarifying:!1})},switchConversation:async n=>{const{isGenerating:a,isClarifying:i,isFixing:s,currentConversationId:c}=t();if(a||i||s||n===c)return;t().saveCurrentConversation(!0),t().clearPendingAttachments();const u=dn(n);if(!u){console.warn(`[AppStore] Conversation not found: ${n}`);return}cn(n),r({currentConversationId:n,messages:u.messages,currentMotion:u.motion,clarifySession:null,isClarifying:!1,renderError:null,isFixing:!1,fixAttemptCount:0})},saveCurrentConversation:(n=!1)=>{bs&&(clearTimeout(bs),bs=null);const a=()=>{const{currentConversationId:i,messages:s,currentMotion:c,conversationList:u}=t();if(!i){if(s.length===0&&!c)return;const C=Cs(),S=Date.now(),P=Ku(s);vo({id:C,title:P,messages:s,motion:c,createdAt:S,updatedAt:S});const T=[{id:C,title:P,updatedAt:S},...u];ln(T),cn(C),r({conversationList:T,currentConversationId:C});return}const p=dn(i),f=u.find(C=>C.id===i),h=Date.now(),x=(p==null?void 0:p.updatedAt)||(f==null?void 0:f.updatedAt)||h,y=(p==null?void 0:p.title)||Ku(s),w={id:i,title:y,messages:s,motion:c,createdAt:(p==null?void 0:p.createdAt)||h,updatedAt:x};vo(w);const E=u.map(C=>C.id===i?{...C,title:y}:C);ln(E),r({conversationList:E})};n?a():bs=setTimeout(a,EC)},deleteConversation:n=>{const{isGenerating:a,isClarifying:i,isFixing:s,currentConversationId:c,conversationList:u}=t();if(n===c&&(a||i||s))return;const p=dn(n);if(p){const h=p.messages.filter(x=>x.attachmentIds&&x.attachmentIds.length>0);h.length>0&&gh(async()=>{const{deleteAttachmentsByMessageId:x}=await Promise.resolve().then(()=>Am);return{deleteAttachmentsByMessageId:x}},[]).then(({deleteAttachmentsByMessageId:x})=>{for(const y of h)x(y.id).catch(w=>{console.error("[AppStore] 删除附件失败:",w)})})}QE(n);const f=u.filter(h=>h.id!==n);if(ln(f),n===c)if(f.length>0){const x=[...f].sort((w,E)=>E.updatedAt-w.updatedAt)[0].id,y=dn(x);cn(x),r({conversationList:f,currentConversationId:x,messages:(y==null?void 0:y.messages)||[],currentMotion:(y==null?void 0:y.motion)||null})}else cn(null),r({conversationList:[],currentConversationId:null,messages:[],currentMotion:null});else r({conversationList:f})},touchConversation:()=>{const{currentConversationId:n,conversationList:a}=t();if(!n)return;const i=Date.now(),s=a.map(c=>c.id===n?{...c,updatedAt:i}:c);s.sort((c,u)=>u.updatedAt-c.updatedAt),ln(s),r({conversationList:s})},toggleHistoryPanel:()=>{r(n=>({isHistoryPanelOpen:!n.isHistoryPanelOpen}))},updateConversationTitle:(n,a)=>{const{conversationList:i}=t();if(!n)return;const s=dn(n);if(s){const u={...s,title:a};vo(u)}const c=i.map(u=>u.id===n?{...u,title:a}:u);ln(c),r({conversationList:c})},duplicateConversation:async n=>{const{isGenerating:a,isClarifying:i,isFixing:s,conversationList:c}=t();if(a||i||s)return null;const u=dn(n);if(!u)return console.warn(`[AppStore] Conversation not found: ${n}`),null;t().saveCurrentConversation(!0);const p=structuredClone(u),f=Cs(),h=Date.now(),x=c.map(S=>S.title),y=wC(u.title,x),w={...p,id:f,title:y,createdAt:h,updatedAt:h};vo(w);const C=[{id:f,title:y,updatedAt:h},...c];return ln(C),cn(f),r({conversationList:C,currentConversationId:f,messages:w.messages,currentMotion:w.motion,clarifySession:null,isClarifying:!1,renderError:null,isFixing:!1,fixAttemptCount:0}),f},importConversations:n=>{const{conversationList:a}=t();if(n.length===0)return[];const i=[],s=[];for(const f of n)vo(f),i.push(f.id),s.push({id:f.id,title:f.title,updatedAt:f.updatedAt});const c=[...s,...a];ln(c);const u=i[0],p=n[0];return cn(u),r({conversationList:c,currentConversationId:u,messages:p.messages,currentMotion:p.motion,clarifySession:null,isClarifying:!1,renderError:null,isFixing:!1,fixAttemptCount:0}),i},initLLMConfigs:async()=>{r({isLoadingConfigs:!0});try{const n=await ZE();if(n){const{configs:i,changed:s}=await im(n.configs);s&&Vn({configs:i,activeConfigId:n.activeConfigId}),r({llmConfigs:i,activeConfigId:n.activeConfigId,isLoadingConfigs:!1});return}const a=Rg();if(a){const{configs:i,changed:s}=await im(a.configs);s&&Vn({configs:i,activeConfigId:a.activeConfigId}),r({llmConfigs:i,activeConfigId:a.activeConfigId,isLoadingConfigs:!1})}else r({llmConfigs:[],activeConfigId:null,isLoadingConfigs:!1})}catch(n){console.error("[AppStore] Failed to init LLM configs:",n),r({llmConfigs:[],activeConfigId:null,isLoadingConfigs:!1})}},setLLMConfigs:n=>{const{activeConfigId:a}=t();r({llmConfigs:n}),Vn({configs:n,activeConfigId:a})},setActiveConfigId:n=>{const{llmConfigs:a}=t();r({activeConfigId:n}),Vn({configs:a,activeConfigId:n})},addLLMConfig:n=>{const{llmConfigs:a,activeConfigId:i}=t(),s=[...a,n],c=a.length===0?n.id:i;r({llmConfigs:s,activeConfigId:c}),Vn({configs:s,activeConfigId:c})},updateLLMConfig:(n,a)=>{const{llmConfigs:i,activeConfigId:s}=t(),c=i.map(p=>p.id===n?{...p,...a,updatedAt:new Date().toISOString()}:p);r({llmConfigs:c}),Vn({configs:c,activeConfigId:s})},deleteLLMConfig:n=>{const{llmConfigs:a,activeConfigId:i}=t(),s=a.filter(p=>p.id!==n);let c=i;i===n&&(c=s.length>0?s[0].id:null),r({llmConfigs:s,activeConfigId:c}),Vn({configs:s,activeConfigId:c})},getActiveConfig:async()=>{const{llmConfigs:n,activeConfigId:a}=t();if(!a)return null;const i=n.find(s=>s.id===a);if(!i)return null;try{const s=Ul(),c=$l(s),u=await Id(i.apiKey,c);return{id:i.id,name:i.name,baseURL:i.baseURL,apiKey:u,model:i.model,createdAt:i.createdAt,updatedAt:i.updatedAt}}catch(s){return console.error("[AppStore] Failed to decrypt active config:",s),null}},openAssetPackExportDialog:()=>{r({isAssetPackExportDialogOpen:!0,isPlaying:!1,assetPackExportState:{...Ns,status:"configuring"}})},closeAssetPackExportDialog:()=>{r({isAssetPackExportDialogOpen:!1,assetPackExportState:Ns})},setAssetPackExportState:n=>{const{assetPackExportState:a}=t();r({assetPackExportState:{...a,...n}})},updateAssetPackExportConfig:n=>{const{assetPackExportState:a}=t(),i=a.config||{filename:"motion-preview",selectedParameterIds:[],showPanelTitle:!0};r({assetPackExportState:{...a,config:{...i,...n}}})},resetAssetPackExportState:()=>{r({assetPackExportState:Ns})},addPendingAttachment:n=>{const{pendingAttachments:a}=t();r({pendingAttachments:[...a,n]})},updatePendingAttachment:(n,a)=>{const{pendingAttachments:i}=t();r({pendingAttachments:i.map(s=>s.tempId===n?{...s,...a}:s)})},removePendingAttachment:n=>{const{pendingAttachments:a}=t(),i=a.find(s=>s.tempId===n);i!=null&&i.previewUrl&&Es(i.previewUrl),r({pendingAttachments:a.filter(s=>s.tempId!==n)})},clearPendingAttachments:()=>{const{pendingAttachments:n}=t();for(const a of n)a.previewUrl&&Es(a.previewUrl);r({pendingAttachments:[]})},loadMessageAttachments:async n=>{const{getAttachmentsByMessageId:a}=await gh(async()=>{const{getAttachmentsByMessageId:i}=await Promise.resolve().then(()=>Am);return{getAttachmentsByMessageId:i}},void 0);return a(n)},setPreviewBackgroundUrl:n=>{const{previewBackgroundUrl:a}=t();a&&a.startsWith("blob:")&&URL.revokeObjectURL(a),r({previewBackgroundUrl:n})},addToast:n=>{const a=`toast_${Date.now()}_${Math.random().toString(36).substring(2,9)}`,i={...n,id:a},{toasts:s}=t();r({toasts:[...s,i]});const c=n.duration??C2;c>0&&setTimeout(()=>{t().removeToast(a)},c)},removeToast:n=>{const{toasts:a}=t();r({toasts:a.filter(i=>i.id!==n)})},setCurrentPath:n=>{r({currentPath:n})},setIsMobileMenuOpen:n=>{r({isMobileMenuOpen:n})},toggleMobileMenu:()=>{const{isMobileMenuOpen:n}=t();r({isMobileMenuOpen:!n})},setLocale:n=>{localStorage.setItem("neon-locale",n),Ct.changeLanguage(n),r({locale:n})}}));function Bd({className:r=""}){const{t}=ft(),n=hn(),{isMobileMenuOpen:a,toggleMobileMenu:i,locale:s,setLocale:c}=Ht(),u=p=>n.pathname===p;return g.jsxs("nav",{className:`fixed top-0 left-0 right-0 z-50 h-16 bg-background-elevated/90 backdrop-blur-md border-b border-border-default ${r}`,children:[g.jsxs("div",{className:"max-w-7xl mx-auto px-4 h-full flex items-center justify-between",children:[g.jsx(Xn,{to:"/",className:"font-display text-xl text-accent-primary hover:opacity-80 transition-opacity cursor-pointer",style:{textShadow:"var(--text-glow)"},children:"Neon"}),g.jsxs("div",{className:"hidden md:flex items-center gap-6",children:[hh.filter(p=>p.showInNav).map(p=>g.jsx(Xn,{to:p.path,className:`font-body text-sm transition-all duration-200 cursor-pointer ${u(p.path)?"text-accent-primary":"text-text-muted hover:text-text-primary"}`,children:t(p.labelKey)},p.path)),g.jsx("button",{onClick:()=>c(s==="zh"?"en":"zh"),className:"px-2 py-1 text-xs font-medium rounded-md border border-border-default text-text-muted hover:text-accent-primary hover:border-accent-primary transition-colors cursor-pointer",title:s==="zh"?"Switch to English":"切换为中文",children:s==="zh"?"EN":"中"})]}),g.jsx("button",{onClick:i,className:"md:hidden p-2 text-text-muted hover:text-text-primary cursor-pointer","aria-label":"Toggle menu",children:g.jsx("svg",{className:"w-6 h-6",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:a?g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M6 18L18 6M6 6l12 12"}):g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M4 6h16M4 12h16M4 18h16"})})})]}),a&&g.jsx("div",{className:"md:hidden absolute top-16 left-0 right-0 bg-background-elevated border-b border-border-default py-4 px-4",children:g.jsxs("div",{className:"flex flex-col gap-4",children:[hh.filter(p=>p.showInNav).map(p=>g.jsx(Xn,{to:p.path,onClick:i,className:`font-body text-sm transition-all duration-200 cursor-pointer ${u(p.path)?"text-accent-primary":"text-text-muted hover:text-text-primary"}`,children:t(p.labelKey)},p.path)),g.jsx("button",{onClick:()=>c(s==="zh"?"en":"zh"),className:"self-start px-2 py-1 text-xs font-medium rounded-md border border-border-default text-text-muted hover:text-accent-primary hover:border-accent-primary transition-colors cursor-pointer",children:s==="zh"?"EN":"中"})]})})]})}const CC=[{id:"neon-lab",title:"Neon Lab",descriptionKey:"nav.neonLab.description",route:"/neon-lab",colorScheme:"green",order:1,status:"available"},{id:"neon-studio",title:"Neon Studio",descriptionKey:"nav.neonStudio.description",route:"/studio",colorScheme:"magenta",order:2,status:"planning"}],Tg=["2026","Aurora","Neon","干杯","BILIBILI"],Xa=1e3,md=2e3,lm=20,Zo={logo:.3,text:.5,normal:.1},bC={gravity:200,friction:.95,particleCount:650,explosionSize:.8,shapeRandomness:.2},dt={textPoints:{},logoPoints:null,lastLogoSrc:null,offscreenCanvas:null};function SC(r){if(!r)return!1;const t=r,n=t.tagName.toUpperCase(),a=t.closest.bind(t);return n==="A"||n==="BUTTON"||n==="INPUT"||n==="SELECT"||n==="TEXTAREA"||a("a")!==null||a("button")!==null||a("[data-no-fireworks]")!==null}function RC(r){var a,i,s,c;let t,n;return"touches"in r?(t=((a=r.touches[0])==null?void 0:a.clientX)??((i=r.changedTouches[0])==null?void 0:i.clientX)??0,n=((s=r.touches[0])==null?void 0:s.clientY)??((c=r.changedTouches[0])==null?void 0:c.clientY)??0):(t=r.clientX,n=r.clientY),{x:t,y:n,timestamp:performance.now()}}function AC(r){if(dt.textPoints[r])return dt.textPoints[r];const t=100;dt.offscreenCanvas||(dt.offscreenCanvas=document.createElement("canvas"));const n=dt.offscreenCanvas.getContext("2d");if(!n)return[];dt.offscreenCanvas.width=t*6,dt.offscreenCanvas.height=t*1.5,n.clearRect(0,0,dt.offscreenCanvas.width,dt.offscreenCanvas.height),n.font=`bold ${t}px "Microsoft YaHei", "PingFang SC", sans-serif`,n.fillStyle="#ffffff",n.textAlign="center",n.textBaseline="middle",n.fillText(r,dt.offscreenCanvas.width/2,t*.75);const a=n.getImageData(0,0,dt.offscreenCanvas.width,dt.offscreenCanvas.height),i=[],s=2;for(let c=0;c<dt.offscreenCanvas.height;c+=s)for(let u=0;u<dt.offscreenCanvas.width;u+=s)a.data[(c*dt.offscreenCanvas.width+u)*4+3]>128&&i.push({x:(u-dt.offscreenCanvas.width/2)/t,y:(c-t*.75)/t});return dt.textPoints[r]=i,i}function PC(r){if(!r||!(r instanceof HTMLImageElement))return null;if(dt.lastLogoSrc===r.src&&dt.logoPoints)return dt.logoPoints;const t=200;dt.offscreenCanvas||(dt.offscreenCanvas=document.createElement("canvas"));const n=dt.offscreenCanvas.getContext("2d");if(!n)return null;dt.offscreenCanvas.width=t,dt.offscreenCanvas.height=t;const a=r.width/r.height;let i=t,s=t;a>1?s=t/a:i=t*a,n.clearRect(0,0,t,t),n.drawImage(r,(t-i)/2,(t-s)/2,i,s);const c=n.getImageData(0,0,t,t),u=[],p=2;for(let f=0;f<t;f+=p)for(let h=0;h<t;h+=p)c.data[(f*t+h)*4+3]>128&&u.push({x:(h-t/2)/t,y:(f-t/2)/t});return dt.lastLogoSrc=r.src,dt.logoPoints=u,u}function kC(r,t){const n=[];for(let a=0;a<t;a++){const i=a/t*Math.PI*2;let s,c;if(r==="star"){const u=1+Math.sin(i*5)*.4;s=u*Math.cos(i-Math.PI/2),c=u*Math.sin(i-Math.PI/2)}else s=16*Math.pow(Math.sin(i),3),c=-(13*Math.cos(i)-5*Math.cos(2*i)-2*Math.cos(3*i)-Math.cos(4*i)),s/=16,c/=16;n.push({x:s,y:c})}return n}function DC(r){return new Promise(t=>{const n=new Image;n.onload=()=>t(n),n.onerror=()=>t(null),n.src=r})}function TC(){return Math.random()>.5?"heart":"star"}let _C=0;function LC(){const r=Math.random();return r<Zo.logo?"logo":r<Zo.logo+Zo.text?"text":r<Zo.logo+Zo.text+Zo.normal?"normal":"shape"}function FC(){return Math.floor(Math.random()*Tg.length)}function NC(){return TC()}function IC(r,t,n,a){const i=LC(),c={id:_C++,x:r/n,targetY:t/a,hue:Math.floor(Math.random()*360),type:i,startTime:performance.now(),textIndex:null,shapeType:null};return i==="text"?c.textIndex=FC():i==="shape"&&(c.shapeType=NC()),c}function cm(r,t,n,a,i,s,c){const p=(1-Math.pow(s,n*60))/Math.max(.001,1-s),f=a+r*p*.05;let h=i+t*p*.05;return h+=.5*c*n*n,{x:f,y:h}}function BC(r,t,n,a,i,s){const c=a-n.startTime;if(c<0||c>Xa+md)return;const{width:u,height:p}=t,{gravity:f,friction:h,particleCount:x,explosionSize:y,shapeRandomness:w}=i;if(c<Xa){const N=c/Xa,I=1-(1-N)*(1-N),L=n.x*u,M=p,H=n.targetY*p,z=M-(M-H)*I,Z=p*.05*(1-N);r.beginPath(),r.moveTo(L,z),r.lineTo(L,z+Z),r.strokeStyle=`hsla(${n.hue}, 100%, 70%, ${1-N})`,r.lineWidth=u*.004,r.lineCap="round",r.stroke(),r.beginPath(),r.arc(L,z,u*.006,0,Math.PI*2),r.fillStyle=`hsl(${n.hue}, 100%, 90%)`,r.fill();return}const E=(c-Xa)/1e3,C=n.x*u,S=n.targetY*p;let b=n.id*1e3;const A=()=>(b=(b*9301+49297)%233280,b/233280);let T=null,B=x,F=u*.3*y;n.type==="logo"&&s?(T=PC(s),T?(B=T.length,F=u*.2*y):n.type="normal"):n.type==="text"&&n.textIndex!==null?(T=AC(Tg[n.textIndex]),T&&(B=T.length,F=u*.3*y)):n.type==="shape"&&n.shapeType&&(B=Math.floor(x*.8),T=kC(n.shapeType,B));const k=Math.ceil(B/1e3);for(let N=0;N<B;N+=k){const I=A(),L=A();let M,H;if(T){const Y=T[N%T.length];M=Y.x*F,H=Y.y*F,M+=(I-.5)*u*.05*w,H+=(L-.5)*u*.05*w}else{const Y=I*Math.PI*2,te=L*F;M=Math.cos(Y)*te,H=Math.sin(Y)*te}const z=1-E/(md/1e3);if(z<=.01)continue;const Z=cm(M,H,E,C,S,h,f*p/1e3),re=Math.max(0,E-.06),se=cm(M,H,re,C,S,h,f*p/1e3);r.beginPath(),r.moveTo(se.x,se.y),r.lineTo(Z.x,Z.y),r.lineWidth=u*.003*2.5,r.strokeStyle=`hsla(${n.hue}, 100%, 60%, ${z*.4})`,r.lineCap="round",r.stroke(),r.beginPath(),r.arc(Z.x,Z.y,u*.003*.8,0,Math.PI*2),r.fillStyle=`hsla(${n.hue}, 100%, 95%, ${z})`,r.fill()}}function jC(r,t,n,a){const{ctx:i,canvas:s}=r;i.globalCompositeOperation="source-over",i.clearRect(0,0,s.width,s.height),i.globalCompositeOperation="lighter";for(const c of t)BC(i,s,c,r.time,n,a);i.globalCompositeOperation="source-over",i.globalAlpha=1}function OC(r,t){return t-r.startTime>Xa+md}function MC(){const r=D.useRef(null),t=D.useRef([]),n=D.useRef(0),a=D.useRef(null),i=D.useRef(0);D.useEffect(()=>{DC("/bilibili-blue.png").then(c=>{a.current=c})},[]);const s=D.useCallback(()=>{const c=r.current;c&&(c.width=window.innerWidth,c.height=window.innerHeight)},[]);return D.useEffect(()=>{const c=r.current;if(!c)return;s(),window.addEventListener("resize",s);const u=f=>{if("touches"in f){const y=performance.now();if(y-i.current<300)return;i.current=y}if(SC(f.target))return;const h=RC(f);t.current.length>=lm&&(t.current=t.current.slice(-lm+1));const x=IC(h.x,h.y,c.width,c.height);t.current.push(x),n.current===0&&(n.current=requestAnimationFrame(p))};document.addEventListener("click",u,!0),document.addEventListener("touchend",u,{passive:!0,capture:!0});const p=()=>{const f=performance.now();t.current=t.current.filter(y=>!OC(y,f));const h=c.getContext("2d");if(!h)return;const x={canvas:c,ctx:h,width:c.width,height:c.height,time:f};jC(x,t.current,bC,a.current),t.current.length>0?n.current=requestAnimationFrame(p):n.current=0};return u.renderLoop=p,()=>{window.removeEventListener("resize",s),document.removeEventListener("click",u,!0),document.removeEventListener("touchend",u,!0),n.current&&(cancelAnimationFrame(n.current),n.current=0)}},[s]),g.jsx("canvas",{ref:r,className:"fixed inset-0 pointer-events-none touch-none",style:{touchAction:"manipulation"},"aria-hidden":"true"})}function $C(){const{t:r}=ft();return g.jsx(jl,{children:g.jsxs("div",{className:"min-h-screen bg-background-primary",children:[g.jsx(Bd,{}),g.jsx(MC,{}),g.jsx("main",{className:"pt-16 px-4 pb-8",children:g.jsxs("div",{className:"max-w-7xl mx-auto",children:[g.jsxs("div",{className:"text-center py-12",children:[g.jsx("h1",{className:"font-display text-4xl md:text-6xl text-accent-primary mb-4",style:{textShadow:"var(--text-glow)"},children:"Neon"}),g.jsx("p",{className:"font-body text-text-muted text-lg md:text-xl",children:r("nav.heroSubtitle")})]}),g.jsx("div",{className:"grid grid-cols-1 md:grid-cols-2 gap-6 max-w-5xl mx-auto",children:CC.map(t=>g.jsx(UC,{entry:t},t.id))})]})})]})})}function UC({entry:r}){const{t}=ft(),n=r.status==="planning",a={cyan:n?"hover:border-accent-primary hover:bg-accent-primary/5 hover:shadow-neon-soft":"hover:border-accent-primary hover:shadow-neon-soft group-hover:shadow-neon-medium",magenta:n?"hover:border-accent-tertiary hover:bg-accent-tertiary/5 hover:shadow-neon-pink":"hover:border-accent-tertiary hover:shadow-neon-pink",purple:n?"hover:border-accent-secondary hover:bg-accent-secondary/5 hover:shadow-neon-purple":"hover:border-accent-secondary hover:shadow-neon-purple",green:n?"hover:border-accent-success hover:bg-accent-success/5 hover:shadow-neon-green":"hover:border-accent-success hover:shadow-neon-green"},i={cyan:"text-accent-primary",magenta:"text-accent-tertiary",purple:"text-accent-secondary",green:"text-accent-success"},s={"neon-lab":"🎬","neon-studio":"🌌"},c=g.jsxs(g.Fragment,{children:[g.jsx("div",{className:`text-5xl mb-4 ${i[r.colorScheme]}`,children:s[r.id]||r.icon||"⚡"}),g.jsx("h3",{className:`font-display text-2xl mb-2 ${n?"text-text-muted":"text-text-primary group-hover:text-accent-primary"} transition-colors`,children:r.title}),g.jsx("p",{className:"font-body text-text-muted text-sm",children:t(r.descriptionKey)}),n&&g.jsx("span",{className:"absolute top-4 right-4 px-2 py-0.5 text-xs font-mono font-bold text-text-muted border border-text-muted/30 rounded bg-text-muted/10",children:"Planning"}),!n&&g.jsx("div",{className:"absolute bottom-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity",children:g.jsx("span",{className:"text-accent-primary text-sm font-body",children:"→"})})]}),u=`group relative min-h-[200px] p-8 rounded-xl border border-border-default bg-gradient-to-br from-background-secondary to-background-elevated transition-all duration-200 ${a[r.colorScheme]}`;return n?g.jsx("div",{className:u,children:c}):g.jsx(Xn,{to:r.route,className:`${u} cursor-pointer hover:scale-[1.02]`,children:c})}function jd(r){return`${"/neon/".replace(/\/+$/,"")}${r}`}const zC="/demos/config.json";let Xu=null;async function HC(r=!1){if(Xu&&!r)return Xu;try{const t=await fetch(jd(zC));if(!t.ok)throw new Error(`Failed to load config: ${t.status} ${t.statusText}`);const n=await t.json();return VC(n),Xu=n,n}catch(t){throw t instanceof Error?new Error(`ConfigLoadError: ${t.message}`):new Error("ConfigLoadError: Unknown error")}}function VC(r){if(!r.version)throw new Error("Invalid config: missing version");if(!Array.isArray(r.items)||r.items.length===0)throw new Error("Invalid config: items must be a non-empty array");if(r.items.length>100)throw new Error("Invalid config: too many items (max 100)");const t=2,n=new Set;function a(s,c){if(s===null)return;if(c>t)throw new Error(`Invalid config: directory depth exceeds ${t}`);if(n.has(s))throw new Error(`Invalid config: circular reference detected at item ${s}`);n.add(s);const u=r.items.find(p=>p.id===s);u&&u.type==="directory"&&a(u.parentId,c+1),n.delete(s)}const i=new Set;for(const s of r.items){if(i.has(s.id))throw new Error(`Invalid config: duplicate item id ${s.id}`);if(i.add(s.id),!/^[a-z0-9][a-z0-9-]*[a-z0-9]$/.test(s.id))throw new Error(`Invalid config: invalid id format ${s.id}`);if(!s.title||s.title.length<1||s.title.length>50)throw new Error(`Invalid config: invalid title for item ${s.id}`);if(s.parentId!==null&&!r.items.some(u=>u.id===s.parentId))throw new Error(`Invalid config: parent directory ${s.parentId} not found for item ${s.id}`);if(s.type==="demo"){const c=s;if(!c.thumbnail||!c.thumbnail.startsWith("/demos/thumbnails/"))throw new Error(`Invalid config: invalid thumbnail path for demo ${s.id}`);const u=/^\/demos\/html\/.+\.html$/;if(!c.htmlPath||!u.test(c.htmlPath))throw new Error(`Invalid config: invalid html path for demo ${s.id}`)}a(s.id,0)}}function WC({demo:r,onClick:t}){const[n,a]=D.useState(!1),[i,s]=D.useState(!1),c=()=>{t(r)},u=()=>{a(!0)},p=()=>{s(!0)},f=r.title.charAt(0).toUpperCase();return g.jsx("button",{onClick:c,className:"group relative w-full cursor-pointer text-left transition-transform duration-200 hover:scale-[1.02] active:scale-[0.98]",style:{textShadow:"var(--glow-pink)"},children:g.jsxs("div",{className:"overflow-hidden rounded-lg bg-surface-primary/50 backdrop-blur-sm ring-1 ring-accent-tertiary/20 transition-shadow duration-200 group-hover:ring-accent-tertiary/50",children:[g.jsxs("div",{className:"relative aspect-video w-full bg-surface-secondary",children:[!i&&!n&&g.jsx("div",{className:"absolute inset-0 animate-pulse bg-surface-secondary"}),n?g.jsx("div",{className:"flex h-full w-full items-center justify-center bg-gradient-to-br from-accent-primary/20 to-accent-secondary/20 text-4xl font-display font-bold text-accent-primary",children:f}):g.jsx("img",{src:jd(r.thumbnail),alt:r.title,className:"h-full w-full object-cover transition-transform duration-300 group-hover:scale-105",onError:u,onLoad:p,loading:"lazy"}),g.jsx("div",{className:"absolute inset-0 flex items-center justify-center bg-black/0 opacity-0 transition-all duration-200 group-hover:bg-black/20 group-hover:opacity-100",children:g.jsx("div",{className:"flex h-12 w-12 items-center justify-center rounded-full bg-accent-primary/90 text-white shadow-lg",children:g.jsx("svg",{className:"h-5 w-5",fill:"currentColor",viewBox:"0 0 20 20",children:g.jsx("path",{d:"M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z"})})})})]}),g.jsxs("div",{className:"p-3",children:[g.jsx("h3",{className:"truncate font-display font-medium text-text-primary group-hover:text-accent-primary",children:r.title}),r.description&&g.jsx("p",{className:"mt-1 line-clamp-2 text-sm text-text-secondary",children:r.description})]})]})})}function GC(){const{t:r}=ft();return g.jsxs("div",{className:"flex min-h-[400px] flex-col items-center justify-center text-center",children:[g.jsx("div",{className:"text-6xl mb-4",children:"🎨"}),g.jsx("h2",{className:"font-display text-2xl text-text-primary mb-2",children:r("demo.empty.title")}),g.jsx("p",{className:"text-text-secondary max-w-md",children:r("demo.empty.descriptionShort")})]})}function qC(){return g.jsx("div",{className:"grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4",children:[...Array(8)].map((r,t)=>g.jsxs("div",{className:"animate-pulse overflow-hidden rounded-lg bg-surface-primary/50 ring-1 ring-accent-tertiary/20",children:[g.jsx("div",{className:"aspect-video bg-surface-secondary"}),g.jsxs("div",{className:"p-3 space-y-2",children:[g.jsx("div",{className:"h-4 bg-surface-secondary rounded w-3/4"}),g.jsx("div",{className:"h-3 bg-surface-secondary/50 rounded w-1/2"})]})]},t))})}function KC({items:r,isLoading:t=!1,onDemoClick:n}){return t?g.jsx(qC,{}):r.length===0?g.jsx(GC,{}):g.jsx("div",{className:"grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4",children:r.map(a=>g.jsx("div",{className:"col-span-1",children:g.jsx(WC,{demo:a,onClick:n})},a.id))})}function XC({demo:r}){const{t}=ft(),[n,a]=D.useState(!0),[i,s]=D.useState(!1),c=D.useRef(null),u=D.useRef(null);D.useEffect(()=>{if(!r){a(!0),s(!1);return}a(!0),s(!1);const h=setTimeout(()=>{a(!1)},1e3);return()=>clearTimeout(h)},[r]);const p=()=>{a(!1),s(!1);const h=c.current;if(h&&h.contentDocument){const x=h.contentDocument.createElement("script");x.textContent=`
        (function() {
          // Listen for height updates from parent
          window.addEventListener('message', function(event) {
            if (event.data && event.data.type === 'SET_CONTAINER_HEIGHT') {
              const height = event.data.height;
              // Set CSS variable for use in styles
              document.documentElement.style.setProperty('--iframe-height', height + 'px');
              // Also directly override vh units
              const style = document.createElement('style');
              style.textContent = \`
                html, body {
                  height: \${height}px !important;
                  min-height: \${height}px !important;
                }
              \`;
              document.head.appendChild(style);
            }
          });
        })();
      `,h.contentDocument.head.appendChild(x),h.contentWindow&&u.current&&h.contentWindow.postMessage({type:"SET_CONTAINER_HEIGHT",height:u.current.clientHeight},"*")}},f=()=>{a(!1),s(!0)};return r?g.jsxs("div",{ref:u,className:"flex h-full flex-col",children:[n&&g.jsx("div",{className:"flex items-center justify-center flex-1 bg-surface-secondary",children:g.jsxs("div",{className:"flex flex-col items-center gap-4",children:[g.jsx("div",{className:"w-10 h-10 border-4 border-accent-primary border-t-transparent rounded-full animate-spin"}),g.jsx("p",{className:"text-text-secondary",children:t("demo.loading")})]})}),i&&g.jsx("div",{className:"flex items-center justify-center flex-1 bg-surface-secondary",children:g.jsxs("div",{className:"text-center",children:[g.jsx("div",{className:"text-4xl mb-4",children:"⚠️"}),g.jsx("h3",{className:"font-display text-lg text-text-primary mb-2",children:t("demo.loadFailed")}),g.jsx("p",{className:"text-text-secondary mb-4",children:t("demo.loadFailedDescription")}),g.jsx("button",{onClick:()=>window.location.reload(),className:"px-4 py-2 rounded-lg bg-accent-primary text-white hover:bg-accent-primary/80 transition-colors",children:t("common.retry")})]})}),!i&&!n&&g.jsx("div",{className:"flex-1 bg-black",style:{height:"100%"},children:g.jsx("iframe",{ref:c,src:jd(r.htmlPath),title:r.title,className:"w-full h-full border-0",sandbox:"allow-scripts allow-same-origin allow-forms allow-popups allow-top-navigation-by-user-activation",onLoad:p,onError:f,allowFullScreen:!0},r.id)})]}):g.jsx("div",{className:"flex h-full items-center justify-center",children:g.jsx("p",{className:"text-text-secondary",children:t("demo.selectHint")})})}const um={BASE_URL:"/neon/",DEV:!1,MODE:"production",PROD:!0,SSR:!1},ai=new Map,Ss=r=>{const t=ai.get(r);return t?Object.fromEntries(Object.entries(t.stores).map(([n,a])=>[n,a.getState()])):{}},YC=(r,t,n)=>{if(r===void 0)return{type:"untracked",connection:t.connect(n)};const a=ai.get(n.name);if(a)return{type:"tracked",store:r,...a};const i={connection:t.connect(n),stores:{}};return ai.set(n.name,i),{type:"tracked",store:r,...i}},QC=(r,t)=>{if(t===void 0)return;const n=ai.get(r);n&&(delete n.stores[t],Object.keys(n.stores).length===0&&ai.delete(r))},JC=r=>{var t,n;if(!r)return;const a=r.split(`
`),i=a.findIndex(c=>c.includes("api.setState"));if(i<0)return;const s=((t=a[i+1])==null?void 0:t.trim())||"";return(n=/.+ (.+) .+/.exec(s))==null?void 0:n[1]},ZC=(r,t={})=>(n,a,i)=>{const{enabled:s,anonymousActionType:c,store:u,...p}=t;let f;try{f=(s??(um?"production":void 0)!=="production")&&window.__REDUX_DEVTOOLS_EXTENSION__}catch{}if(!f)return r(n,a,i);const{connection:h,...x}=YC(u,f,p);let y=!0;i.setState=((C,S,P)=>{const b=n(C,S);if(!y)return b;const A=P===void 0?{type:c||JC(new Error().stack)||"anonymous"}:typeof P=="string"?{type:P}:P;return u===void 0?(h==null||h.send(A,a()),b):(h==null||h.send({...A,type:`${u}/${A.type}`},{...Ss(p.name),[u]:i.getState()}),b)}),i.devtools={cleanup:()=>{h&&typeof h.unsubscribe=="function"&&h.unsubscribe(),QC(p.name,u)}};const w=(...C)=>{const S=y;y=!1,n(...C),y=S},E=r(i.setState,a,i);if(x.type==="untracked"?h==null||h.init(E):(x.stores[x.store]=i,h==null||h.init(Object.fromEntries(Object.entries(x.stores).map(([C,S])=>[C,C===x.store?E:S.getState()])))),i.dispatchFromDevtools&&typeof i.dispatch=="function"){let C=!1;const S=i.dispatch;i.dispatch=(...P)=>{(um?"production":void 0)!=="production"&&P[0].type==="__setState"&&!C&&(console.warn('[zustand devtools middleware] "__setState" action type is reserved to set state from the devtools. Avoid using it.'),C=!0),S(...P)}}return h.subscribe(C=>{var S;switch(C.type){case"ACTION":if(typeof C.payload!="string"){console.error("[zustand devtools middleware] Unsupported action format");return}return Yu(C.payload,P=>{if(P.type==="__setState"){if(u===void 0){w(P.state);return}Object.keys(P.state).length!==1&&console.error(`
                    [zustand devtools middleware] Unsupported __setState action format.
                    When using 'store' option in devtools(), the 'state' should have only one key, which is a value of 'store' that was passed in devtools(),
                    and value of this only key should be a state object. Example: { "type": "__setState", "state": { "abc123Store": { "foo": "bar" } } }
                    `);const b=P.state[u];if(b==null)return;JSON.stringify(i.getState())!==JSON.stringify(b)&&w(b);return}i.dispatchFromDevtools&&typeof i.dispatch=="function"&&i.dispatch(P)});case"DISPATCH":switch(C.payload.type){case"RESET":return w(E),u===void 0?h==null?void 0:h.init(i.getState()):h==null?void 0:h.init(Ss(p.name));case"COMMIT":if(u===void 0){h==null||h.init(i.getState());return}return h==null?void 0:h.init(Ss(p.name));case"ROLLBACK":return Yu(C.state,P=>{if(u===void 0){w(P),h==null||h.init(i.getState());return}w(P[u]),h==null||h.init(Ss(p.name))});case"JUMP_TO_STATE":case"JUMP_TO_ACTION":return Yu(C.state,P=>{if(u===void 0){w(P);return}JSON.stringify(i.getState())!==JSON.stringify(P[u])&&w(P[u])});case"IMPORT_STATE":{const{nextLiftedState:P}=C.payload,b=(S=P.computedStates.slice(-1)[0])==null?void 0:S.state;if(!b)return;w(u===void 0?b:b[u]),h==null||h.send(null,P);return}case"PAUSE_RECORDING":return y=!y}return}}),E},eb=ZC,Yu=(r,t)=>{let n;try{n=JSON.parse(r)}catch(a){console.error("[zustand devtools middleware] Could not parse the received json",a)}n!==void 0&&t(n)},ii=mg()(eb(r=>({currentDirectoryId:null,selectedDemoId:null,viewMode:"grid",isLoading:!1,error:null,items:[],navigationHistory:{history:[],currentIndex:-1},scrollPosition:0,setItems:t=>r({items:t}),setCurrentDirectory:t=>r({currentDirectoryId:t,viewMode:"grid",selectedDemoId:null}),setSelectedDemo:t=>r({selectedDemoId:t,viewMode:t?"detail":"grid"}),setViewMode:t=>r({viewMode:t}),pushToHistory:t=>r(n=>{const a=[...n.navigationHistory.history.slice(0,n.navigationHistory.currentIndex+1),t];return{navigationHistory:{history:a,currentIndex:a.length-1}}}),goBack:()=>r(t=>{if(t.navigationHistory.currentIndex<=0)return{currentDirectoryId:null,navigationHistory:{history:[],currentIndex:-1}};const n=t.navigationHistory.currentIndex-1;return{currentDirectoryId:t.navigationHistory.history[n],navigationHistory:{...t.navigationHistory,currentIndex:n}}}),setLoading:t=>r({isLoading:t}),setError:t=>r({error:t}),clearError:()=>r({error:null}),setScrollPosition:t=>r({scrollPosition:t})}),{name:"DemoGalleryStore"})),tb=()=>{const r=ii(t=>t.items);return D.useMemo(()=>r.filter(t=>t.type==="demo"&&t.parentId===null).sort((t,n)=>t.order-n.order),[r])},_g=()=>{const r=ii(n=>n.items),t=ii(n=>n.selectedDemoId);return D.useMemo(()=>t?r.find(n=>n.id===t&&n.type==="demo"):null,[r,t])};function rb({onBack:r}){const{t}=ft(),n=_g(),{setSelectedDemo:a}=ii(),i=()=>{a(null);const s=new URL(window.location.href);s.searchParams.delete("id"),window.history.pushState({},"",s.toString()),r==null||r()};return g.jsx("nav",{className:"sticky top-16 z-40 w-full border-b border-accent-tertiary/20 bg-background-primary/80 backdrop-blur-md",children:g.jsx("div",{className:"max-w-7xl mx-auto px-4",children:g.jsxs("div",{className:"flex items-center gap-4 h-14",children:[g.jsxs("button",{onClick:i,className:"flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg hover:bg-surface-secondary transition-colors text-text-primary hover:text-accent-primary","aria-label":t("common.back"),children:[g.jsx("svg",{className:"w-4 h-4",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M15 19l-7-7 7-7"})}),g.jsx("span",{className:"text-sm font-medium",children:t("common.back")})]}),g.jsx("div",{className:"flex-1"}),n&&g.jsxs("div",{className:"hidden sm:block text-sm text-text-secondary",children:[t("demo.viewing"),g.jsx("span",{className:"text-text-primary",children:n.title})]})]})})})}function nb(){const[r]=i2(),t=r.get("id"),{isLoading:n,error:a,viewMode:i,setSelectedDemo:s,setError:c,setItems:u,setLoading:p,scrollPosition:f,setScrollPosition:h}=ii(),x=tb(),y=_g(),w=D.useRef(null);D.useEffect(()=>{(async()=>{p(!0),c(null);try{const P=await HC();u(P.items)}catch(P){c(P instanceof Error?P.message:"Failed to load demos")}finally{p(!1)}})()},[]),D.useEffect(()=>{s(t||null)},[t]),D.useEffect(()=>{!t&&f>0&&w.current&&i==="grid"&&(w.current.scrollTop=f)},[t,f,i]),D.useEffect(()=>{const S=()=>{w.current&&!t&&i==="grid"&&h(w.current.scrollTop)},P=w.current;return P==null||P.addEventListener("scroll",S),()=>P==null?void 0:P.removeEventListener("scroll",S)},[t,i]),D.useEffect(()=>{const S=()=>{new URLSearchParams(window.location.search).get("id")||s(null)};return window.addEventListener("popstate",S),()=>window.removeEventListener("popstate",S)},[]);const E=S=>{s(S.id);const P=new URL(window.location.href);P.searchParams.set("id",S.id),window.history.pushState({},"",P.toString())},C=()=>{s(null);const S=new URL(window.location.href);S.searchParams.delete("id"),window.history.pushState({},"",S.toString())};return g.jsx(jl,{children:g.jsxs("div",{className:"min-h-screen bg-background-primary",children:[g.jsx(Bd,{}),i==="detail"&&y&&g.jsxs("div",{className:"fixed inset-0 z-50 pt-16",children:[g.jsx(rb,{onBack:C}),g.jsx("div",{className:"h-[calc(100vh-8rem)]",children:g.jsx(XC,{demo:y})})]}),g.jsx("main",{ref:w,className:`pt-24 px-4 pb-8 transition-all ${i==="detail"?"opacity-0 pointer-events-none":""}`,children:g.jsxs("div",{className:"max-w-7xl mx-auto",children:[a&&g.jsx("div",{className:"mb-8 rounded-lg bg-red-500/10 border border-red-500/30 p-4",children:g.jsxs("div",{className:"flex items-center gap-3",children:[g.jsx("span",{className:"text-2xl",children:"⚠️"}),g.jsxs("div",{children:[g.jsx("h3",{className:"font-display font-medium text-text-primary",children:"加载失败"}),g.jsx("p",{className:"text-sm text-text-secondary mt-1",children:a})]}),g.jsx("button",{onClick:()=>window.location.reload(),className:"ml-auto px-4 py-2 rounded-lg bg-accent-primary/20 hover:bg-accent-primary/30 text-accent-primary transition-colors",children:"重试"})]})}),!a&&g.jsxs(g.Fragment,{children:[g.jsxs("div",{className:"mb-8 mt-6",children:[g.jsx("h1",{className:"font-display text-3xl md:text-4xl text-text-primary mb-2",children:"效果Demo"}),g.jsx("p",{className:"font-display text-accent-tertiary",style:{textShadow:"var(--glow-pink)"},children:"探索各种精彩的动画效果Demo"})]}),g.jsx(KC,{items:x,isLoading:n,onDemoClick:E})]})]})})]})})}const ob={primary:"bg-accent-primary text-black hover:bg-accent-primary/90 disabled:opacity-50 shadow-neon-soft",secondary:"bg-transparent text-accent-primary border border-accent-primary hover:bg-accent-primary/10 disabled:opacity-50 disabled:border-accent-primary/30",ghost:"bg-transparent text-text-primary hover:bg-accent-primary/10 hover:text-accent-primary disabled:opacity-50 disabled:text-text-muted",danger:"bg-accent-tertiary text-white hover:bg-accent-tertiary/90 disabled:opacity-50 shadow-neon-pink"},ab={sm:"px-3 py-1.5 text-sm min-h-[36px]",md:"px-4 py-2 text-sm min-h-[40px]",lg:"px-6 py-3 text-base min-h-[44px]"},lt=D.forwardRef(({variant:r="primary",size:t="md",loading:n=!1,disabled:a,className:i="",children:s,...c},u)=>g.jsxs("button",{ref:u,disabled:a||n,className:`
          inline-flex items-center justify-center
          rounded-lg font-medium font-body
          transition-all duration-200 cursor-pointer
          focus:outline-none focus:ring-2 focus:ring-accent-primary focus:ring-offset-2 focus:ring-offset-background-primary
          disabled:cursor-not-allowed
          ${ob[r]}
          ${ab[t]}
          ${i}
        `.trim(),...c,children:[n&&g.jsxs("svg",{className:"animate-spin -ml-1 mr-2 h-4 w-4",xmlns:"http://www.w3.org/2000/svg",fill:"none",viewBox:"0 0 24 24",children:[g.jsx("circle",{className:"opacity-25",cx:"12",cy:"12",r:"10",stroke:"currentColor",strokeWidth:"4"}),g.jsx("path",{className:"opacity-75",fill:"currentColor",d:"M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"})]}),s]}));lt.displayName="Button";const wo=D.forwardRef(({label:r,error:t,helperText:n,className:a="",id:i,...s},c)=>{const u=i||(r==null?void 0:r.toLowerCase().replace(/\s+/g,"-"));return g.jsxs("div",{className:"w-full",children:[r&&g.jsx("label",{htmlFor:u,className:"block text-sm font-medium font-body text-text-primary mb-1",children:r}),g.jsx("input",{ref:c,id:u,className:`
            w-full px-3 py-2
            border border-border-default rounded-lg
            bg-background-elevated text-text-primary placeholder:text-text-muted
            focus:outline-none focus:ring-2 focus:ring-accent-primary focus:border-transparent
            disabled:opacity-50 disabled:cursor-not-allowed
            transition-colors duration-200
            ${t?"border-accent-tertiary":""}
            ${a}
          `.trim(),...s}),t&&g.jsx("p",{className:"mt-1 text-sm font-body text-accent-tertiary",children:t}),n&&!t&&g.jsx("p",{className:"mt-1 text-sm font-body text-text-muted",children:n})]})});wo.displayName="Input";const Lg=D.forwardRef(({label:r,min:t=0,max:n=100,step:a=1,value:i=0,showValue:s=!0,unit:c="",className:u="",id:p,onChange:f,...h},x)=>{const y=p||(r==null?void 0:r.toLowerCase().replace(/\s+/g,"-"));return g.jsxs("div",{className:"w-full",children:[g.jsxs("div",{className:"flex items-center justify-between mb-1",children:[r&&g.jsx("label",{htmlFor:y,className:"text-sm font-medium font-body text-text-primary",children:r}),s&&g.jsxs("span",{className:"text-sm font-mono text-text-muted",children:[i,c]})]}),g.jsx("input",{ref:x,id:y,type:"range",min:t,max:n,step:a,value:i,onChange:f,className:`
            w-full h-2
            bg-border-default rounded-lg
            appearance-none cursor-pointer
            accent-accent-primary
            focus:outline-none focus:ring-2 focus:ring-accent-primary focus:ring-offset-2 focus:ring-offset-background-primary
            disabled:opacity-50 disabled:cursor-not-allowed
            ${u}
          `.trim(),...h}),g.jsxs("div",{className:"flex justify-between text-xs text-text-muted mt-1",children:[g.jsxs("span",{children:[t,c]}),g.jsxs("span",{children:[n,c]})]})]})});Lg.displayName="Slider";const Fg=D.forwardRef(({label:r,value:t="#000000",className:n="",id:a,onChange:i,...s},c)=>{const u=a||(r==null?void 0:r.toLowerCase().replace(/\s+/g,"-"));return g.jsxs("div",{className:"w-full",children:[r&&g.jsx("label",{htmlFor:u,className:"block text-sm font-medium font-body text-text-primary mb-1",children:r}),g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx("div",{className:"relative",children:g.jsx("input",{ref:c,id:u,type:"color",value:t,onChange:i,className:`
                w-10 h-10
                border border-border-default rounded-lg cursor-pointer
                focus:outline-none focus:ring-2 focus:ring-accent-primary
                disabled:opacity-50 disabled:cursor-not-allowed
                ${n}
              `.trim(),...s})}),g.jsx("span",{className:"text-sm font-mono uppercase text-text-muted",children:t})]})]})});Fg.displayName="ColorPicker";const kl=D.forwardRef(({label:r,options:t,error:n,placeholder:a,className:i="",id:s,value:c,onChange:u,...p},f)=>{const h=s||(r==null?void 0:r.toLowerCase().replace(/\s+/g,"-"));return g.jsxs("div",{className:"w-full",children:[r&&g.jsx("label",{htmlFor:h,className:"block text-sm font-medium font-body text-text-primary mb-1",children:r}),g.jsxs("select",{ref:f,id:h,value:c,onChange:u,className:`
            w-full px-3 py-2
            border border-border-default rounded-lg
            text-text-primary bg-background-elevated
            focus:outline-none focus:ring-2 focus:ring-accent-primary focus:border-transparent
            disabled:opacity-50 disabled:cursor-not-allowed
            transition-colors duration-200
            ${n?"border-accent-tertiary":""}
            ${i}
          `.trim(),...p,children:[a&&g.jsx("option",{value:"",disabled:!0,children:a}),t.map(x=>g.jsx("option",{value:x.value,children:x.label},x.value))]}),n&&g.jsx("p",{className:"mt-1 text-sm font-body text-accent-tertiary",children:n})]})});kl.displayName="Select";const Od=D.forwardRef(({label:r,checked:t=!1,className:n="",id:a,onChange:i,...s},c)=>{const u=a||(r==null?void 0:r.toLowerCase().replace(/\s+/g,"-"));return g.jsxs("div",{className:"flex items-center justify-between",children:[r&&g.jsx("label",{htmlFor:u,className:"text-sm font-medium font-body text-text-primary cursor-pointer",children:r}),g.jsx("button",{type:"button",role:"switch","aria-checked":t,onClick:()=>{i&&i({target:{checked:!t}})},className:`
            relative inline-flex h-6 w-11
            shrink-0 cursor-pointer
            rounded-full border-2 border-transparent
            transition-colors duration-200 ease-in-out
            focus:outline-none focus:ring-2 focus:ring-accent-primary focus:ring-offset-2 focus:ring-offset-background-primary
            disabled:opacity-50 disabled:cursor-not-allowed
            ${t?"bg-accent-primary":"bg-border-default"}
            ${n}
          `.trim(),disabled:s.disabled,children:g.jsx("span",{className:`
              pointer-events-none inline-block h-5 w-5
              transform rounded-full bg-white shadow-lg
              transition duration-200 ease-in-out
              ${t?"translate-x-5":"translate-x-0"}
            `.trim()})}),g.jsx("input",{ref:c,id:u,type:"checkbox",checked:t,onChange:i,className:"sr-only",...s})]})});Od.displayName="Toggle";function Ng({isOpen:r,title:t,message:n,confirmLabel:a="确认",cancelLabel:i="取消",onConfirm:s,onCancel:c,variant:u="default"}){const p=D.useRef(null);D.useEffect(()=>{const h=x=>{x.key==="Escape"&&r&&c()};return r&&document.addEventListener("keydown",h),()=>{document.removeEventListener("keydown",h)}},[r,c]);const f=h=>{h.target===h.currentTarget&&c()};return r?g.jsx("div",{className:"fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm",onClick:f,children:g.jsxs("div",{ref:p,className:"bg-background-elevated rounded-lg shadow-neon-soft max-w-sm w-full mx-4 overflow-hidden border border-border-default",role:"dialog","aria-modal":"true","aria-labelledby":"confirm-dialog-title",children:[g.jsxs("div",{className:"p-4",children:[g.jsx("h3",{id:"confirm-dialog-title",className:"text-lg font-medium font-display text-text-primary mb-2",children:t}),g.jsx("p",{className:"text-sm font-body text-text-muted",children:n})]}),g.jsxs("div",{className:"flex justify-end gap-2 px-4 py-3 bg-background-secondary border-t border-border-default",children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:c,children:i}),g.jsx(lt,{variant:u==="danger"?"danger":"primary",size:"sm",onClick:s,children:a})]})]})}):null}function ib({toast:r,onClose:t}){const n=D.useCallback(()=>{t(r.id)},[r.id,t]),a={warning:"bg-amber-500 text-white",error:"bg-red-500 text-white",info:"bg-blue-500 text-white",success:"bg-green-500 text-white"},i={warning:"⚠️",error:"❌",info:"ℹ️",success:"✓"};return g.jsxs("div",{className:`
        flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg
        min-w-[280px] max-w-[400px]
        animate-slide-in-right
        ${a[r.type]}
      `,role:"alert",children:[g.jsx("span",{className:"text-lg flex-shrink-0",children:i[r.type]}),g.jsx("p",{className:"flex-1 text-sm font-medium",children:r.message}),g.jsx("button",{onClick:n,className:"flex-shrink-0 p-1 hover:bg-white/20 rounded transition-colors","aria-label":"关闭",children:g.jsx("svg",{className:"w-4 h-4",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M6 18L18 6M6 6l12 12"})})})]})}function sb({toasts:r,onClose:t}){return r.length===0?null:g.jsx("div",{className:"fixed top-4 right-4 z-50 flex flex-col gap-2","aria-live":"polite",children:r.map(n=>g.jsx(ib,{toast:n,onClose:t},n.id))})}const Yn={SYSTEM_LOGS:"motion-platform:system-logs",CODE_LOGS:"motion-platform:code-logs"},oa={MAX_SYSTEM_LOGS:1e3,MAX_CODE_LOGS:100,MAX_AGE_DAYS:7,MAX_RESPONSE_SIZE:50*1024};function Ig(){return Math.random().toString(36).substring(2,6)}function Bg(){return`log_${Date.now()}_${Ig()}`}function yl(){return`req_${Date.now()}_${Ig()}`}function jg(r,t){if(!r)return t;try{return JSON.parse(r)}catch{return t}}function Og(r){try{return localStorage.getItem(r)}catch{return null}}function Md(r,t){try{return localStorage.setItem(r,t),!0}catch{return!1}}function $d(r){try{localStorage.removeItem(r)}catch{}}function Ud(){const r=Og(Yn.SYSTEM_LOGS);return jg(r,[])}function lb(r){const t=Ud(),n={...r,id:Bg(),timestamp:Date.now()};t.push(n);const a=cb(t);Md(Yn.SYSTEM_LOGS,JSON.stringify(a))||zg(Yn.SYSTEM_LOGS,a)}function Mg(){$d(Yn.SYSTEM_LOGS)}function zd(){const r=Og(Yn.CODE_LOGS);return jg(r,[])}function $g(r){const t=zd(),a={...db(r),id:Bg(),timestamp:Date.now()};t.push(a);const i=ub(t);Md(Yn.CODE_LOGS,JSON.stringify(i))||zg(Yn.CODE_LOGS,i)}function Ug(){$d(Yn.CODE_LOGS)}function cb(r){const t=Date.now(),n=oa.MAX_AGE_DAYS*24*60*60*1e3;let a=r.filter(i=>t-i.timestamp<n);return a.length>oa.MAX_SYSTEM_LOGS&&(a=a.slice(-1e3)),a}function ub(r){const t=Date.now(),n=oa.MAX_AGE_DAYS*24*60*60*1e3;let a=r.filter(i=>t-i.timestamp<n);return a.length>oa.MAX_CODE_LOGS&&(a=a.slice(-100)),a}function db(r){return r.response.length<=oa.MAX_RESPONSE_SIZE?r:{...r,response:r.response.substring(0,oa.MAX_RESPONSE_SIZE)+`

[TRUNCATED - Response exceeded 50KB limit]`,truncated:!0}}function zg(r,t){const n=Math.floor(t.length/2),a=t.slice(-n);Md(r,JSON.stringify(a))||($d(r),console.warn(`[Logger] Cleared ${r} due to quota exceeded`))}function fb(){Mg(),Ug()}function Hg(r,t,n){const a=dm(r,n),i=dm(t,n),c=[...a,...i].map(u=>u.timestamp).sort((u,p)=>u-p);return{exportedAt:Date.now(),systemLogs:a,codeLogs:i,metadata:{totalSystemLogs:a.length,totalCodeLogs:i.length,oldestLogTime:c.length>0?c[0]:null,newestLogTime:c.length>0?c[c.length-1]:null}}}function dm(r,t){return!(t!=null&&t.startTime)&&!(t!=null&&t.endTime)?r:r.filter(n=>!(t.startTime&&n.timestamp<t.startTime||t.endTime&&n.timestamp>t.endTime))}function pb(r,t,n){const a=Hg(r,t,n),i=JSON.stringify(a,null,2),s=new Blob([i],{type:"application/json"}),u=`motion-platform-logs-${new Date().toISOString().replace(/[:.]/g,"-")}.json`;hb(s,u)}function hb(r,t){const n=URL.createObjectURL(r),a=document.createElement("a");a.href=n,a.download=t,document.body.appendChild(a),a.click(),document.body.removeChild(a),URL.revokeObjectURL(n)}function Rs(r,t,n,a){try{lb({level:r,module:t,message:n,metadata:a});const i=mb(r),s=`[${r}][${t}]`;a?i(s,n,a):i(s,n)}catch{}}function mb(r){switch(r){case"DEBUG":return console.debug;case"INFO":return console.info;case"WARN":return console.warn;case"ERROR":return console.error;default:return console.log}}function gb(r,t,n,a){try{$g({level:"INFO",module:"LLMService",message:"代码生成完成",requestId:r,codeType:"generation",prompt:t,response:n,extractedCode:a})}catch{}}function xb(r,t,n,a,i,s){try{$g({level:"INFO",module:"LLMService",message:"代码修复完成",requestId:r,codeType:"fix",prompt:t,response:n,extractedCode:a,originalCode:i,errorMessage:s})}catch{}}function vb(r){try{return Hg(Ud(),zd(),r)}catch{return{exportedAt:Date.now(),systemLogs:[],codeLogs:[],metadata:{totalSystemLogs:0,totalCodeLogs:0,oldestLogTime:null,newestLogTime:null}}}}function yb(r){try{pb(Ud(),zd(),r)}catch{}}function wb(){try{Mg()}catch{}}function Eb(){try{Ug()}catch{}}function Cb(){try{fb()}catch{}}const pe={debug:(r,t,n)=>Rs("DEBUG",r,t,n),info:(r,t,n)=>Rs("INFO",r,t,n),warn:(r,t,n)=>Rs("WARN",r,t,n),error:(r,t,n)=>Rs("ERROR",r,t,n),logCodeGeneration:gb,logCodeFix:xb,export:vb,exportToFile:yb,clearSystemLogs:wb,clearCodeLogs:Eb,clearAll:Cb},Tl=class Tl{async syncVideoToTime(t,n,a={}){const{timeout:i=Tl.DEFAULT_TIMEOUT,loop:s=!0,startTimeOffset:c=0}=a,u=n-c,p=Math.max(0,u)/1e3,f=t.duration||1;let h;if(s?(h=p%f,h===0&&p>0&&(h=f-.001)):h=Math.min(p,f-.001),!(Math.abs(t.currentTime-h)<.01))return t.paused||t.pause(),new Promise(x=>{const y=()=>{t.removeEventListener("seeked",y),x()};t.addEventListener("seeked",y),t.currentTime=h,setTimeout(()=>{t.removeEventListener("seeked",y),x()},i)})}async syncAllVideosToTime(t,n,a={},i={}){const s=Object.entries(t).filter(([,c])=>c instanceof HTMLVideoElement);s.length!==0&&await Promise.all(s.map(([c,u])=>{const p=i[c]??0;return this.syncVideoToTime(u,n,{...a,startTimeOffset:p})}))}async cloneVideoForExport(t,n=5e3){return new Promise((a,i)=>{const s=document.createElement("video");s.src=t.src,s.muted=!0,s.preload="auto";const c=()=>{s.removeEventListener("loadeddata",u),s.removeEventListener("error",p)},u=()=>{c(),clearTimeout(f),pe.debug("VideoSyncService","视频克隆成功",{src:t.src}),a(s)},p=()=>{c(),clearTimeout(f),pe.error("VideoSyncService","视频克隆失败",{src:t.src}),i(new Error("视频克隆失败"))};s.addEventListener("loadeddata",u,{once:!0}),s.addEventListener("error",p,{once:!0});const f=setTimeout(()=>{c(),pe.warn("VideoSyncService","视频克隆超时",{src:t.src}),i(new Error("视频克隆超时"))},n)})}async cloneAllVideosForExport(t){const n={},a=Object.entries(t).filter(([,i])=>i instanceof HTMLVideoElement);for(const[i,s]of a)try{n[i]=await this.cloneVideoForExport(s)}catch(c){pe.warn("VideoSyncService",`跳过克隆失败的视频: ${i}`,{error:c instanceof Error?c.message:String(c)}),n[i]=s}return n}disposeClonedVideos(t){for(const n of Object.values(t))n.pause(),n.src="",n.load();pe.debug("VideoSyncService","已释放克隆视频资源",{count:Object.keys(t).length})}};ve(Tl,"DEFAULT_TIMEOUT",200);let gd=Tl;const Wn=new gd;class bb{getParameterValue(t){var n,a;switch(t.type){case"number":return t.value??t.min??0;case"color":return t.colorValue??"#000000";case"select":return t.selectedValue??((a=(n=t.options)==null?void 0:n[0])==null?void 0:a.value)??"";case"boolean":return t.boolValue??!1;case"image":return t.imageValue??t.placeholderImage??Td;case"video":return t.videoValue??t.placeholderVideo??_d;case"string":return t.stringValue??"";default:return null}}initializeParams(t){const n={};return t.parameters.forEach(a=>{n[a.id]=this.getParameterValue(a)}),n}async loadImageForParam(t){if(Pg(t)){const n=Al();return vl(n)}return vl(t)}async preloadImageParams(t,n){const a=t.parameters.filter(s=>s.type==="image"),i=[];for(const s of a){const c=n[s.id];if(c)try{const u=await this.loadImageForParam(c);n[s.id]=u,i.push({id:s.id,value:u,success:!0})}catch(u){pe.warn("ParameterLoaderService",`加载图片参数失败: ${s.id}`,{error:u instanceof Error?u.message:String(u)});try{const p=Al(),f=await vl(p);n[s.id]=f,i.push({id:s.id,value:f,success:!1,error:u instanceof Error?u.message:String(u)})}catch(p){i.push({id:s.id,value:null,success:!1,error:`加载占位图也失败: ${p}`})}}}return i}async loadVideoForParam(t){if(kg(t)){const n=Pl();return qu(n)}return qu(t)}async preloadVideoParams(t,n){const a=t.parameters.filter(s=>s.type==="video"),i=[];for(const s of a){const c=n[s.id];if(c)try{const u=await this.loadVideoForParam(c);n[s.id]=u,i.push({id:s.id,value:u,success:!0})}catch(u){pe.warn("ParameterLoaderService",`加载视频参数失败: ${s.id}`,{error:u instanceof Error?u.message:String(u)});try{const p=Pl(),f=await qu(p);n[s.id]=f,i.push({id:s.id,value:f,success:!1,error:u instanceof Error?u.message:String(u)})}catch(p){n[s.id]=null,i.push({id:s.id,value:null,success:!1,error:`加载占位视频也失败: ${p}`})}}}return i}async preloadAllMediaParams(t,n){const[a,i]=await Promise.all([this.preloadImageParams(t,n),this.preloadVideoParams(t,n)]);return[...a,...i]}}const As=new bb;function Gl(r){let t=r>>>0;return function(){t=t+1831565813>>>0;let a=t;return a=Math.imul(a^a>>>15,a|1),a^=a+Math.imul(a^a>>>7,a|61),((a^a>>>14)>>>0)/4294967296}}function Sb(r,t,n){const a=Gl(r);return Math.floor(a()*(n-t+1))+t}function Rb(r,t,n){return Gl(r)()*(n-t)+t}function Ab(r){return Gl(r)}function Pb(){return{seededRandom:Gl,seededRandomInt:Sb,seededRandomRange:Rb,createRandomSequence:Ab}}const Vg=`
var VideoStartTimeEvaluator = {
  /**
   * 从参数数组构建评估上下文
   * 返回简化的上下文结构：
   * - params.numberParam → 数值
   * - params.videoParam.videoDuration → 视频时长（毫秒）
   */
  buildParamsContext: function(parameters, runtimeParams) {
    var context = {};
    for (var i = 0; i < parameters.length; i++) {
      var param = parameters[i];
      var runtimeValue = runtimeParams ? runtimeParams[param.id] : undefined;

      switch (param.type) {
        case 'number':
          context[param.id] = typeof runtimeValue === 'number' ? runtimeValue : (param.value || 0);
          break;
        case 'color':
          context[param.id] = typeof runtimeValue === 'string' ? runtimeValue : (param.colorValue || '#000000');
          break;
        case 'boolean':
          context[param.id] = typeof runtimeValue === 'boolean' ? runtimeValue : (param.boolValue || false);
          break;
        case 'select':
          context[param.id] = typeof runtimeValue === 'string' ? runtimeValue : (param.selectedValue || '');
          break;
        case 'video':
          var videoContext = {};
          if (runtimeValue && runtimeValue.tagName === 'VIDEO') {
            videoContext.videoDuration = (runtimeValue.duration || 0) * 1000;
            videoContext.videoWidth = runtimeValue.videoWidth || 0;
            videoContext.videoHeight = runtimeValue.videoHeight || 0;
          } else {
            videoContext.videoDuration = param.videoDuration || 0;
            videoContext.videoWidth = param.videoWidth || 0;
            videoContext.videoHeight = param.videoHeight || 0;
          }
          videoContext.videoStartTime = param.videoStartTime || 0;
          context[param.id] = videoContext;
          break;
      }
    }
    return context;
  },

  /**
   * 安全执行动态计算代码
   */
  evaluateCode: function(code, paramsContext) {
    if (!code || code.trim() === '') {
      return { value: 0, success: true };
    }

    try {
      var fn = new Function('params', 'Math', '"use strict"; return (' + code + ');');
      var result = fn(paramsContext, Math);

      if (typeof result !== 'number' || !isFinite(result)) {
        return { value: 0, success: false, error: '计算结果不是有效数值: ' + result };
      }

      return { value: Math.max(0, result), success: true };
    } catch (e) {
      console.warn('[VideoStartTimeEvaluator] 代码执行失败:', e.message);
      return { value: 0, success: false, error: e.message };
    }
  },

  /**
   * 获取视频参数的有效起始时间
   */
  getEffectiveStartTime: function(param, parameters, runtimeParams) {
    if (param.type !== 'video') return 0;

    if (param.videoStartTimeCode && param.videoStartTimeCode.trim() !== '') {
      var context = this.buildParamsContext(parameters, runtimeParams);
      var result = this.evaluateCode(param.videoStartTimeCode, context);
      if (result.success) {
        return result.value;
      }
      console.warn('[VideoStartTimeEvaluator] 参数', param.id, '的动态计算失败:', result.error);
    }

    return Math.max(0, param.videoStartTime || 0);
  },

  /**
   * 获取所有视频参数的起始时间映射
   */
  getAllVideoStartTimes: function(parameters, runtimeParams) {
    var result = {};
    for (var i = 0; i < parameters.length; i++) {
      var param = parameters[i];
      if (param.type === 'video') {
        result[param.id] = this.getEffectiveStartTime(param, parameters, runtimeParams);
      }
    }
    return result;
  },

  /**
   * 计算动态时长 (025-dynamic-duration)
   * @param {string} code - 动态计算代码
   * @param {object} paramsContext - 参数上下文
   * @param {number} fallbackDuration - 降级默认值
   * @returns {{ value: number, success: boolean, error?: string }}
   */
  evaluateDuration: function(code, paramsContext, fallbackDuration) {
    var MIN_DURATION = 1000;
    var MAX_DURATION = 60000;

    if (!code || code.trim() === '') {
      return { value: fallbackDuration, success: true };
    }

    var result = this.evaluateCode(code, paramsContext);

    if (!result.success) {
      console.warn('[DynamicEvaluator] Duration 计算失败，使用降级值:', result.error);
      return { value: fallbackDuration, success: false, error: result.error };
    }

    var duration = result.value;

    // 边界校验：处理 NaN、Infinity、负数
    if (!isFinite(duration) || duration < 0) {
      console.warn('[DynamicEvaluator] Duration 为无效值，使用降级值:', duration);
      return { value: fallbackDuration, success: false, error: '计算结果为无效值: ' + duration };
    }

    // 边界校验：最小值
    if (duration < MIN_DURATION) {
      console.warn('[DynamicEvaluator] Duration 低于最小值，使用 1000ms');
      duration = MIN_DURATION;
    }

    // 边界校验：最大值
    if (duration > MAX_DURATION) {
      console.warn('[DynamicEvaluator] Duration 超过最大值，使用 60000ms');
      duration = MAX_DURATION;
    }

    return { value: duration, success: true };
  },

  /**
   * 获取有效的动效时长 (025-dynamic-duration)
   * @param {object} motion - MotionDefinition (需要包含 duration, durationCode, parameters)
   * @param {object} runtimeParams - 运行时参数值
   * @returns {number} 有效时长（毫秒）
   */
  getEffectiveDuration: function(motion, runtimeParams) {
    if (!motion) {
      return 5000; // 默认 5 秒
    }

    if (!motion.durationCode || motion.durationCode.trim() === '') {
      return motion.duration || 5000;
    }

    var context = this.buildParamsContext(motion.parameters || [], runtimeParams);
    var result = this.evaluateDuration(motion.durationCode, context, motion.duration || 5000);
    return result.value;
  }
};
`.trim(),kb=()=>new Function(Vg+`
return VideoStartTimeEvaluator;`)(),Wg=kb();function Db(r,t){return Wg.getAllVideoStartTimes(r,t)}function Tb(r,t){return Wg.getEffectiveDuration(r,t)}let _b=0;class fm{constructor(t={}){ve(this,"_container",null);ve(this,"motion",null);ve(this,"canvas",null);ve(this,"ctx",null);ve(this,"_isPlaying",!1);ve(this,"startTime",0);ve(this,"pausedTime",0);ve(this,"animationFrameId",null);ve(this,"renderFunction",null);ve(this,"params",{});ve(this,"instanceId");ve(this,"actualWidth",640);ve(this,"actualHeight",360);ve(this,"onError");ve(this,"hasErrored",!1);ve(this,"currentCode","");ve(this,"videoStartTimeOffsets",{});ve(this,"effectiveDuration",0);ve(this,"onDurationChange");ve(this,"onPerformanceWarning");ve(this,"tick",async()=>{if(!this._isPlaying||!this.motion)return;let t=performance.now()-this.startTime;const n=this.effectiveDuration||this.motion.duration;t>=n&&(this.startTime=performance.now(),t=0),await Wn.syncAllVideosToTime(this.params,t,{},this.videoStartTimeOffsets);const a=performance.now();this.renderFrame(t);const i=performance.now()-a;if(i>Wu&&this.onPerformanceWarning){pe.warn("CanvasRenderer","单帧渲染超时",{elapsed:i,threshold:Wu}),this.onPerformanceWarning({elapsed:i,threshold:Wu,timestamp:Date.now()});return}this.animationFrameId=requestAnimationFrame(this.tick)});this.instanceId=`renderer_${++_b}_${Date.now()}`,this.onError=t.onError,this.onDurationChange=t.onDurationChange,this.onPerformanceWarning=t.onPerformanceWarning}async initialize(t,n){pe.info("CanvasRenderer","初始化",{instanceId:this.instanceId,motionId:n.id}),this._container=t,this.motion=n,this.cleanup(),this.hasErrored=!1,this.currentCode=n.code,this.actualWidth=n.width,this.actualHeight=n.height,pe.debug("CanvasRenderer","Canvas dimensions",{width:this.actualWidth,height:this.actualHeight}),this.canvas=document.createElement("canvas"),this.canvas.width=this.actualWidth,this.canvas.height=this.actualHeight,this.canvas.style.display="block",t.appendChild(this.canvas),this.ctx=this.canvas.getContext("2d"),this.initializeParams(),this.loadRenderFunction(n.code),await this.preloadMediaParams(),this.updateVideoStartTimeOffsets(),this.updateDynamicDuration(),await Wn.syncAllVideosToTime(this.params,0,{},this.videoStartTimeOffsets),this.renderFrame(0)}async initializeOffscreen(t,n){if(this.motion=t,this.canvas=n,this.ctx=n.getContext("2d"),!this.ctx)throw new Error("无法获取 2D 上下文");this.actualWidth=n.width,this.actualHeight=n.height,this.hasErrored=!1,this.currentCode=t.code,this.initializeParams(),this.loadRenderFunction(t.code),await this.preloadMediaParams(),this.updateVideoStartTimeOffsets(),this.updateDynamicDuration()}resize(t,n){!this.canvas||this.actualWidth===t&&this.actualHeight===n||(this.canvas.width=t,this.canvas.height=n,this.actualWidth=t,this.actualHeight=n)}updateVideoStartTimeOffsets(){this.motion&&(this.videoStartTimeOffsets=Db(this.motion.parameters,this.params),Object.keys(this.videoStartTimeOffsets).length>0&&pe.debug("CanvasRenderer","视频起始时间偏移已计算",this.videoStartTimeOffsets))}updateDynamicDuration(){if(!this.motion)return;const t=this.effectiveDuration;this.effectiveDuration=Tb(this.motion,this.params),pe.debug("CanvasRenderer","动态时长已计算",{effectiveDuration:this.effectiveDuration,hasDurationCode:!!this.motion.durationCode,durationCode:this.motion.durationCode,fixedDuration:this.motion.duration,runtimeParams:Object.keys(this.params).reduce((n,a)=>{const i=this.params[a];return n[a]=i instanceof HTMLElement?`[${i.tagName}]`:i,n},{})}),t!==this.effectiveDuration&&t!==0&&(this.onDurationChange?(this.onDurationChange(this.effectiveDuration),pe.info("CanvasRenderer","时长变化回调已触发",{previousDuration:t,newDuration:this.effectiveDuration})):pe.warn("CanvasRenderer","时长变化但未设置 onDurationChange 回调",{previousDuration:t,newDuration:this.effectiveDuration}))}initializeParams(){this.motion&&(this.params=As.initializeParams(this.motion))}async preloadMediaParams(){this.motion&&await As.preloadAllMediaParams(this.motion,this.params)}loadRenderFunction(t){if(!t){this.renderFunction=null;return}try{window.__motionUtils=Pb(),window.__motionRender=void 0,window.__motionRenders||(window.__motionRenders={}),delete window.__motionRenders[this.instanceId];const n=this.instanceId,a=`
        (function() {
          ${t}
          if (typeof window.__motionRender === 'function') {
            window.__motionRenders['${n}'] = window.__motionRender;
            window.__motionRender = undefined;
          }
        })();
      `;new Function(a)(),typeof window.__motionRenders[n]=="function"?(this.renderFunction=window.__motionRenders[n],pe.debug("CanvasRenderer","Render function loaded",{instanceKey:n})):(pe.warn("CanvasRenderer","Motion code did not define render function"),this.renderFunction=null)}catch(n){if(pe.error("CanvasRenderer","Error loading render function",{error:n instanceof Error?n.message:String(n)}),this.renderFunction=null,this.onError&&n instanceof Error){const a=this.createRenderError("syntax",n);pe.warn("CanvasRenderer","Reporting syntax error",{message:a.message}),this.onError(a),this.hasErrored=!0}}}createRenderError(t,n){return{id:vg(),type:t,message:n.message,friendlyMessage:xg(n.name),lineNumber:n.lineNumber,columnNumber:n.columnNumber,code:this.currentCode,timestamp:Date.now()}}renderFrame(t){if(!this.ctx||!this.canvas||!this.motion)return;this.ctx.clearRect(0,0,this.canvas.width,this.canvas.height),this.motion.backgroundColor&&this.motion.backgroundColor!=="transparent"&&(this.ctx.fillStyle=this.motion.backgroundColor,this.ctx.fillRect(0,0,this.canvas.width,this.canvas.height));const n={width:this.actualWidth,height:this.actualHeight};if(this.renderFunction)try{this.ctx.save(),this.renderFunction(this.ctx,t,this.params,n),this.ctx.restore()}catch(a){if(this.hasErrored)pe.debug("CanvasRenderer","Interval render error (not reporting to UI)",{error:a instanceof Error?a.message:String(a)});else if(pe.error("CanvasRenderer","First-frame runtime error",{error:a instanceof Error?a.message:String(a)}),this.onError&&a instanceof Error){const i=this.createRenderError("runtime",a);pe.warn("CanvasRenderer","Reporting runtime error",{message:i.message}),this.onError(i),this.hasErrored=!0}}else this.ctx.save(),this.renderElements(t),this.ctx.restore()}renderElements(t){if(!this.ctx||!this.motion){pe.debug("CanvasRenderer","renderElements: ctx 或 motion 为空");return}if(!this.motion.elements||!Array.isArray(this.motion.elements)){pe.debug("CanvasRenderer","renderElements: elements 无效");return}this.motion.elements.forEach((n,a)=>{if(!n||!n.properties){pe.debug("CanvasRenderer",`元素 ${a} 无效`);return}const i=n.properties,s=n.animation||{duration:this.motion.duration,delay:0,loop:!0},c=s.duration||this.motion.duration,u=s.delay||0,p=Math.max(0,t-u);let f=p%c/c;!s.loop&&p>=c&&(f=1);const h=this.interpolateKeyframes(i,s.keyframes,f);this.ctx.save();const x=typeof h.x=="number"?h.x:i.x||0,y=typeof h.y=="number"?h.y:i.y||0;this.ctx.translate(x,y);const w=typeof h.rotation=="number"?h.rotation:i.rotation||0;w!==0&&this.ctx.rotate(w*Math.PI/180);const E=typeof h.opacity=="number"?h.opacity:i.opacity??1;this.ctx.globalAlpha=E,n.type==="shape"?this.drawShape(h):n.type==="text"&&h.text&&this.drawText(h),this.ctx.restore()})}interpolateKeyframes(t,n,a){const i={...t};if(!n||n.length===0)return i;let s=n[0],c=n[n.length-1];for(let h=0;h<n.length-1;h++)if(a>=n[h].offset&&a<=n[h+1].offset){s=n[h],c=n[h+1];break}const u=c.offset-s.offset,p=u>0?(a-s.offset)/u:0,f={...i};return Object.keys(c.properties).forEach(h=>{const x=s.properties,y=c.properties,w=x[h]??i[h],E=y[h];typeof w=="number"&&typeof E=="number"?f[h]=w+(E-w)*p:f[h]=E}),f}drawShape(t){if(!this.ctx)return;const n=t.width||100,a=t.height||100,i=t.shape||"rectangle";if(this.ctx.beginPath(),i==="circle"){const s=Math.min(n,a)/2;this.ctx.arc(0,0,s,0,Math.PI*2)}else if(i==="triangle")this.ctx.moveTo(0,-a/2),this.ctx.lineTo(n/2,a/2),this.ctx.lineTo(-n/2,a/2),this.ctx.closePath();else{const s=t.borderRadius||0;s>0?this.roundRect(-n/2,-a/2,n,a,s):this.ctx.rect(-n/2,-a/2,n,a)}t.fill&&(this.ctx.fillStyle=t.fill,this.ctx.fill()),t.stroke&&(this.ctx.strokeStyle=t.stroke,this.ctx.lineWidth=t.strokeWidth||1,this.ctx.stroke())}roundRect(t,n,a,i,s){this.ctx&&(this.ctx.beginPath(),this.ctx.moveTo(t+s,n),this.ctx.lineTo(t+a-s,n),this.ctx.quadraticCurveTo(t+a,n,t+a,n+s),this.ctx.lineTo(t+a,n+i-s),this.ctx.quadraticCurveTo(t+a,n+i,t+a-s,n+i),this.ctx.lineTo(t+s,n+i),this.ctx.quadraticCurveTo(t,n+i,t,n+i-s),this.ctx.lineTo(t,n+s),this.ctx.quadraticCurveTo(t,n,t+s,n),this.ctx.closePath())}drawText(t){if(!this.ctx)return;const n=t.text,a=t.fontSize||16,i=t.fontFamily||"sans-serif",s=t.color||"#000000";this.ctx.font=`${a}px ${i}`,this.ctx.fillStyle=s,this.ctx.textAlign="center",this.ctx.textBaseline="middle",this.ctx.fillText(n,0,0)}play(){this._isPlaying||(this._isPlaying=!0,this.startTime=performance.now()-this.pausedTime,this.tick())}pause(){this._isPlaying&&(this._isPlaying=!1,this.pausedTime=performance.now()-this.startTime,this.animationFrameId&&(cancelAnimationFrame(this.animationFrameId),this.animationFrameId=null))}stop(){this.pause(),this.pausedTime=0,this.renderFrame(0)}async seek(t){this.pausedTime=t,this.startTime=performance.now()-t,await Wn.syncAllVideosToTime(this.params,t,{},this.videoStartTimeOffsets),this.renderFrame(t)}async updateParameter(t,n){var i;const a=(i=this.motion)==null?void 0:i.parameters.find(s=>s.id===t);if((a==null?void 0:a.type)==="image"&&typeof n=="string")try{const s=await As.loadImageForParam(n);this.params[t]=s,this._isPlaying||this.renderFrame(this.pausedTime)}catch(s){pe.warn("CanvasRenderer",`更新图片参数失败: ${t}`,{error:s instanceof Error?s.message:String(s)})}else if((a==null?void 0:a.type)==="video"&&typeof n=="string")try{const s=await As.loadVideoForParam(n);this.params[t]=s,s.muted=!0,this.updateVideoStartTimeOffsets(),this.updateDynamicDuration(),this._isPlaying||(await Wn.syncAllVideosToTime(this.params,this.pausedTime,{},this.videoStartTimeOffsets),this.renderFrame(this.pausedTime))}catch(s){pe.warn("CanvasRenderer",`更新视频参数失败: ${t}`,{error:s instanceof Error?s.message:String(s)})}else this.params[t]=n,this.updateVideoStartTimeOffsets(),this.updateDynamicDuration(),this._isPlaying||this.renderFrame(this.pausedTime)}updateDuration(t){this.motion&&(this.motion={...this.motion,duration:t},pe.debug("CanvasRenderer","动效时长已更新",{duration:t}))}getCurrentTime(){return this._isPlaying?performance.now()-this.startTime:this.pausedTime}getCanvas(){return this.canvas}getDimensions(){return{width:this.actualWidth,height:this.actualHeight}}isPlaying(){return this._isPlaying}getDuration(){var t;return this.effectiveDuration||((t=this.motion)==null?void 0:t.duration)||0}getParameters(){return{...this.params}}async renderAt(t,n={}){const{waitForVideoSeek:a=!0,videoOverrides:i}=n,s=i?{...this.params,...i}:this.params;a&&await Wn.syncAllVideosToTime(s,t,{},this.videoStartTimeOffsets),this.renderFrameWithParams(t,s)}async renderToCanvas(t,n,a={}){const{waitForVideoSeek:i=!0,videoOverrides:s}=a,c=s?{...this.params,...s}:this.params;i&&await Wn.syncAllVideosToTime(c,n,{},this.videoStartTimeOffsets);const u=t.getContext("2d");if(!u||!this.motion)throw new Error("无法获取目标 canvas context 或动效未初始化");const p=this.ctx,f=this.actualWidth,h=this.actualHeight;try{this.ctx=u,this.actualWidth=t.width,this.actualHeight=t.height,this.renderFrameWithParams(n,c)}finally{this.ctx=p,this.actualWidth=f,this.actualHeight=h}}renderFrameWithParams(t,n){if(!this.ctx||!this.motion)return;this.ctx.clearRect(0,0,this.actualWidth,this.actualHeight),this.motion.backgroundColor&&this.motion.backgroundColor!=="transparent"&&(this.ctx.fillStyle=this.motion.backgroundColor,this.ctx.fillRect(0,0,this.actualWidth,this.actualHeight));const a={width:this.actualWidth,height:this.actualHeight};if(this.renderFunction)try{this.ctx.save(),this.renderFunction(this.ctx,t,n,a),this.ctx.restore()}catch(i){pe.debug("CanvasRenderer","renderFrameWithParams error",{error:i instanceof Error?i.message:String(i)})}else this.ctx.save(),this.renderElements(t),this.ctx.restore()}getWebGLContext(){return null}destroy(){this.cleanup(),this._container=null,this.motion=null}cleanup(){this.animationFrameId&&(cancelAnimationFrame(this.animationFrameId),this.animationFrameId=null),this.canvas&&(this.canvas.remove(),this.canvas=null),this.ctx=null,this.renderFunction=null,this.params={},this._isPlaying=!1,this.startTime=0,this.pausedTime=0,window.__motionRenders&&this.instanceId&&delete window.__motionRenders[this.instanceId]}}const Lb=`
attribute vec2 aPosition;
attribute vec2 aTexCoord;
varying vec2 vUv;

void main() {
  vUv = aTexCoord;
  gl_Position = vec4(aPosition, 0.0, 1.0);
}
`,Fb=`
precision highp float;
uniform sampler2D uTexture;
varying vec2 vUv;

void main() {
  gl_FragColor = texture2D(uTexture, vUv);
}
`;function Nb(r){let t=0;for(let n=0;n<r.length;n++){const a=r.charCodeAt(n);t=(t<<5)-t+a,t=t&t}return t.toString(36)}function Ib(r){const t=[],n=[];if(/uniform\s+sampler2D\s+uTexture/.test(r)||t.push("uniform sampler2D uTexture;"),/uniform\s+sampler2D\s+uOriginal/.test(r)||t.push("uniform sampler2D uOriginal;"),/uniform\s+vec2\s+uResolution/.test(r)||t.push("uniform vec2 uResolution;"),/uniform\s+float\s+uTime/.test(r)||t.push("uniform float uTime;"),/varying\s+vec2\s+vUv/.test(r)||n.push("varying vec2 vUv;"),t.length===0&&n.length===0)return r.includes("precision ")?r:`precision highp float;

`+r;let a=r;r.includes("precision ")&&(a=r.replace(/precision\s+(lowp|mediump|highp)\s+float\s*;/g,""));const i=["precision highp float;"];return i.push(...t),i.push(...n),i.join(`
`)+`

`+a}function Bb(r,t){const n=[];for(const[a,i]of Object.entries(t)){const s=Array.isArray(i)?`vec${i.length}`:"float";new RegExp(`uniform\\s+\\w+\\s+${a}\\b`).test(r)||n.push(`uniform ${s} ${a};`)}return n.length===0?r:n.join(`
`)+`
`+r}class jb{constructor(t){ve(this,"canvas");ve(this,"gl",null);ve(this,"disabled",!1);ve(this,"sourceTexture",null);ve(this,"pingPongTextures",[null,null]);ve(this,"pingPongFramebuffers",[null,null]);ve(this,"programCache",new Map);ve(this,"passthroughProgram",null);ve(this,"vertexBuffer",null);ve(this,"texCoordBuffer",null);ve(this,"postProcessFn",null);ve(this,"onError");ve(this,"postProcessCode","");ve(this,"hasReportedError",!1);ve(this,"width",0);ve(this,"height",0);this.options=t,this.onError=t==null?void 0:t.onError,this.canvas=null}initialize(t,n,a){var s;this.width=n,this.height=a,this.canvas=document.createElement("canvas"),this.canvas.width=n,this.canvas.height=a,this.canvas.style.display="block",(s=this.options)!=null&&s.exportMode||t.appendChild(this.canvas);const i=this.canvas.getContext("webgl",{alpha:!0,premultipliedAlpha:!1,preserveDrawingBuffer:!0})||this.canvas.getContext("experimental-webgl");return i?(this.gl=i,this.initBuffers(),this.initTextures(),this.initPassthroughProgram(),!0):(console.warn("[PostProcess] WebGL 不可用，后处理已禁用"),this.disabled=!0,!1)}loadPostProcessFunction(t){if(!t||!t.trim()){this.postProcessFn=null,this.postProcessCode="";return}this.postProcessCode=t,this.hasReportedError=!1,this.programCache.clear();try{const n=`postProcess_${Date.now()}_${Math.random().toString(36).slice(2)}`;window.__motionPostProcesses||(window.__motionPostProcesses={});const a=`
        (function() {
          ${t}
          if (typeof window.__motionPostProcess === 'function') {
            window.__motionPostProcesses['${n}'] = window.__motionPostProcess;
            window.__motionPostProcess = undefined;
          }
        })();
      `;new Function(a)();const s=window.__motionPostProcesses;this.postProcessFn=s[n]||null,delete s[n],this.postProcessFn||console.warn("[PostProcess] 代码未定义 window.__motionPostProcess 函数")}catch(n){console.error("[PostProcess] 加载后处理函数失败:",n),this.postProcessFn=null,this.reportError(n instanceof Error?n.message:String(n))}}render(t,n,a){if(!this.canvas)return;if(this.disabled||!this.gl){this.fallbackRender(t);return}const i=this.gl;this.uploadSourceTexture(t);let s=[];if(this.postProcessFn)try{s=this.postProcessFn(a,n)}catch(f){console.error("[PostProcess] 执行 postProcess 函数失败:",f),s=[]}if(s.length===0){this.renderPassthrough();return}let c=this.sourceTexture,u=0,p=!1;for(let f=0;f<s.length;f++){const h=s[f],x=f===s.length-1;let y=h.shader;h.uniforms&&(y=Bb(y,h.uniforms));const w=this.getOrCompileProgram(h.name,y);if(!w)continue;x?(i.bindFramebuffer(i.FRAMEBUFFER,null),i.viewport(0,0,this.width,this.height),p=!0):(i.bindFramebuffer(i.FRAMEBUFFER,this.pingPongFramebuffers[u]),i.viewport(0,0,this.width,this.height)),i.clearColor(0,0,0,0),i.clear(i.COLOR_BUFFER_BIT),i.useProgram(w.program),i.activeTexture(i.TEXTURE0),i.bindTexture(i.TEXTURE_2D,c),i.activeTexture(i.TEXTURE1),i.bindTexture(i.TEXTURE_2D,this.sourceTexture);const E=w.uniforms.get("uTexture");E&&i.uniform1i(E,0);const C=w.uniforms.get("uOriginal");C&&i.uniform1i(C,1);const S=w.uniforms.get("uResolution");S&&i.uniform2f(S,this.width,this.height);const P=w.uniforms.get("uTime");if(P&&i.uniform1f(P,n),h.uniforms)for(const[b,A]of Object.entries(h.uniforms)){const T=w.uniforms.get(b);if(T)if(Array.isArray(A))switch(A.length){case 2:i.uniform2fv(T,A);break;case 3:i.uniform3fv(T,A);break;case 4:i.uniform4fv(T,A);break}else i.uniform1f(T,A)}this.bindVertexAttributes(w.program),i.drawArrays(i.TRIANGLE_STRIP,0,4),x||(c=this.pingPongTextures[u],u=1-u)}p||this.renderPassthrough(c)}resize(t,n){t===this.width&&n===this.height||(this.width=t,this.height=n,this.canvas.width=t,this.canvas.height=n,this.gl&&this.initTextures())}getCanvas(){return this.canvas}dispose(){var t,n;if(this.gl){const a=this.gl;this.sourceTexture&&a.deleteTexture(this.sourceTexture),this.pingPongTextures.forEach(i=>i&&a.deleteTexture(i)),this.pingPongFramebuffers.forEach(i=>i&&a.deleteFramebuffer(i)),this.vertexBuffer&&a.deleteBuffer(this.vertexBuffer),this.texCoordBuffer&&a.deleteBuffer(this.texCoordBuffer),this.programCache.forEach(i=>{i&&a.deleteProgram(i.program)}),this.passthroughProgram&&a.deleteProgram(this.passthroughProgram.program)}(n=(t=this.canvas)==null?void 0:t.parentElement)==null||n.removeChild(this.canvas),this.gl=null,this.programCache.clear()}reportError(t){if(this.hasReportedError||!this.onError)return;this.hasReportedError=!0;const n={id:vg(),type:"syntax",message:t,friendlyMessage:xg("SyntaxError"),code:this.postProcessCode,source:"postProcess",timestamp:Date.now()};this.onError(n)}initBuffers(){const t=this.gl,n=new Float32Array([-1,-1,1,-1,-1,1,1,1]);this.vertexBuffer=t.createBuffer(),t.bindBuffer(t.ARRAY_BUFFER,this.vertexBuffer),t.bufferData(t.ARRAY_BUFFER,n,t.STATIC_DRAW);const a=new Float32Array([0,0,1,0,0,1,1,1]);this.texCoordBuffer=t.createBuffer(),t.bindBuffer(t.ARRAY_BUFFER,this.texCoordBuffer),t.bufferData(t.ARRAY_BUFFER,a,t.STATIC_DRAW)}initTextures(){const t=this.gl;this.sourceTexture&&t.deleteTexture(this.sourceTexture),this.pingPongTextures.forEach(n=>n&&t.deleteTexture(n)),this.pingPongFramebuffers.forEach(n=>n&&t.deleteFramebuffer(n)),this.sourceTexture=this.createTexture();for(let n=0;n<2;n++){const a=this.createTexture(),i=t.createFramebuffer();if(t.bindFramebuffer(t.FRAMEBUFFER,i),t.framebufferTexture2D(t.FRAMEBUFFER,t.COLOR_ATTACHMENT0,t.TEXTURE_2D,a,0),t.checkFramebufferStatus(t.FRAMEBUFFER)!==t.FRAMEBUFFER_COMPLETE){console.warn("[PostProcess] Framebuffer not complete, disabling post-process"),this.disabled=!0;return}this.pingPongTextures[n]=a,this.pingPongFramebuffers[n]=i}t.bindFramebuffer(t.FRAMEBUFFER,null)}createTexture(){const t=this.gl,n=t.createTexture();return t.bindTexture(t.TEXTURE_2D,n),t.texImage2D(t.TEXTURE_2D,0,t.RGBA,this.width,this.height,0,t.RGBA,t.UNSIGNED_BYTE,null),t.texParameteri(t.TEXTURE_2D,t.TEXTURE_MIN_FILTER,t.LINEAR),t.texParameteri(t.TEXTURE_2D,t.TEXTURE_MAG_FILTER,t.LINEAR),t.texParameteri(t.TEXTURE_2D,t.TEXTURE_WRAP_S,t.CLAMP_TO_EDGE),t.texParameteri(t.TEXTURE_2D,t.TEXTURE_WRAP_T,t.CLAMP_TO_EDGE),n}initPassthroughProgram(){this.passthroughProgram=this.compileProgram("__passthrough__",Fb)}uploadSourceTexture(t){const n=this.gl;n.bindTexture(n.TEXTURE_2D,this.sourceTexture),n.pixelStorei(n.UNPACK_FLIP_Y_WEBGL,!0),n.texImage2D(n.TEXTURE_2D,0,n.RGBA,n.RGBA,n.UNSIGNED_BYTE,t)}getOrCompileProgram(t,n){const a=`${t}_${Nb(n)}`;if(this.programCache.has(a))return this.programCache.get(a)??null;const i=this.compileProgram(a,n);return this.programCache.set(a,i),i||this.reportError(`Shader "${t}" compilation failed`),i}compileProgram(t,n){const a=this.gl,i=Ib(n),s=a.createShader(a.VERTEX_SHADER);if(a.shaderSource(s,Lb),a.compileShader(s),!a.getShaderParameter(s,a.COMPILE_STATUS))return console.error("[PostProcess] Vertex shader 编译失败:",a.getShaderInfoLog(s)),a.deleteShader(s),null;const c=a.createShader(a.FRAGMENT_SHADER);if(a.shaderSource(c,i),a.compileShader(c),!a.getShaderParameter(c,a.COMPILE_STATUS)){const h=a.getShaderInfoLog(c)||"Unknown shader error";return console.error(`[PostProcess] Fragment shader "${t}" 编译失败:`,h),this.reportError(`Fragment shader "${t}" 编译失败: ${h}`),a.deleteShader(s),a.deleteShader(c),null}const u=a.createProgram();if(a.attachShader(u,s),a.attachShader(u,c),a.linkProgram(u),!a.getProgramParameter(u,a.LINK_STATUS)){const h=a.getProgramInfoLog(u)||"Unknown link error";return console.error("[PostProcess] 程序链接失败:",h),this.reportError(`Program link failed: ${h}`),a.deleteProgram(u),a.deleteShader(s),a.deleteShader(c),null}a.deleteShader(s),a.deleteShader(c);const p=new Map,f=a.getProgramParameter(u,a.ACTIVE_UNIFORMS);for(let h=0;h<f;h++){const x=a.getActiveUniform(u,h);x&&p.set(x.name,a.getUniformLocation(u,x.name))}return{program:u,uniforms:p}}bindVertexAttributes(t){const n=this.gl,a=n.getAttribLocation(t,"aPosition");a>=0&&(n.bindBuffer(n.ARRAY_BUFFER,this.vertexBuffer),n.enableVertexAttribArray(a),n.vertexAttribPointer(a,2,n.FLOAT,!1,0,0));const i=n.getAttribLocation(t,"aTexCoord");i>=0&&(n.bindBuffer(n.ARRAY_BUFFER,this.texCoordBuffer),n.enableVertexAttribArray(i),n.vertexAttribPointer(i,2,n.FLOAT,!1,0,0))}renderPassthrough(t){if(!this.passthroughProgram||!this.gl)return;const n=this.gl;n.bindFramebuffer(n.FRAMEBUFFER,null),n.viewport(0,0,this.width,this.height),n.clearColor(0,0,0,0),n.clear(n.COLOR_BUFFER_BIT),n.useProgram(this.passthroughProgram.program),n.activeTexture(n.TEXTURE0),n.bindTexture(n.TEXTURE_2D,t||this.sourceTexture);const a=this.passthroughProgram.uniforms.get("uTexture");a&&n.uniform1i(a,0),this.bindVertexAttributes(this.passthroughProgram.program),n.drawArrays(n.TRIANGLE_STRIP,0,4)}fallbackRender(t){const n=this.canvas.getContext("2d");n&&n.drawImage(t,0,0)}}class Ob{constructor(t,n){ve(this,"canvasRenderer");ve(this,"postProcessor");ve(this,"offscreenCanvas");ve(this,"motion",null);ve(this,"params",{});ve(this,"_isPlaying",!1);ve(this,"startTime",0);ve(this,"pausedTime",0);ve(this,"animationFrameId",null);ve(this,"_isDestroyed",!1);ve(this,"tick",()=>{if(!this._isPlaying||!this.motion)return;let t=performance.now()-this.startTime;const n=this.getDuration();t>=n&&(this.startTime=performance.now(),t=0),this.render(t),this.animationFrameId=requestAnimationFrame(this.tick)});this.canvasRenderer=t,this.postProcessor=n,this.offscreenCanvas=document.createElement("canvas")}async initialize(t,n){this.motion=n,this._isDestroyed=!1,this.offscreenCanvas.width=n.width,this.offscreenCanvas.height=n.height,this.params=n.parameters.reduce((a,i)=>(a[i.id]=i.value,a),{}),n.postProcessCode&&this.postProcessor.loadPostProcessFunction(n.postProcessCode),await this.canvasRenderer.initializeOffscreen(n,this.offscreenCanvas),!this._isDestroyed&&this.postProcessor.initialize(t,n.width,n.height)}render(t){this.motion&&(this.canvasRenderer.renderAt(t),this.postProcessor.render(this.offscreenCanvas,t,this.params))}renderToCanvas(t,n){if(!this.motion)return;this.canvasRenderer.renderAt(n),this.postProcessor.render(this.offscreenCanvas,n,this.params);const a=t.getContext("2d");a&&a.drawImage(this.postProcessor.getCanvas(),0,0)}updateParameter(t,n){this.canvasRenderer.updateParameter(t,n),this.params[t]=n}getCurrentTime(){return this._isPlaying?performance.now()-this.startTime:this.pausedTime}getDuration(){return this.canvasRenderer.getDuration()}getDimensions(){return{width:this.offscreenCanvas.width,height:this.offscreenCanvas.height}}getParameters(){return{...this.params}}isPlaying(){return this._isPlaying}async renderAt(t){this.render(t)}play(){this._isPlaying||(this._isPlaying=!0,this.startTime=performance.now()-this.pausedTime,this.tick())}pause(){this._isPlaying&&(this._isPlaying=!1,this.pausedTime=performance.now()-this.startTime,this.animationFrameId&&(cancelAnimationFrame(this.animationFrameId),this.animationFrameId=null))}stop(){this.pause(),this.pausedTime=0}seek(t){this.pausedTime=t,this._isPlaying&&(this.startTime=performance.now()-t),this.render(t)}resize(t,n){this.offscreenCanvas.width=t,this.offscreenCanvas.height=n,this.postProcessor.resize(t,n),this.canvasRenderer.resize(t,n)}getCanvas(){return this.postProcessor.getCanvas()}destroy(){this._isDestroyed=!0,this.pause(),this.canvasRenderer.destroy(),this.postProcessor.dispose()}dispose(){this.destroy()}updateDuration(t){var n,a;(a=(n=this.canvasRenderer).updateDuration)==null||a.call(n,t)}}function Gg(r,t){if(console.log("[RendererFactory] motion.postProcessCode:",r.postProcessCode?`存在 (${r.postProcessCode.length} chars)`:"不存在"),r.postProcessCode&&r.postProcessCode.trim().length>0){console.log("[RendererFactory] 使用 CanvasWithPostProcessRenderer (后处理模式)");const a=new fm(t),i=new jb({exportMode:t==null?void 0:t.exportMode,onError:t==null?void 0:t.onError});return new Ob(a,i)}return new fm(t)}function Mb(r,t){const[n,a]=D.useState(r);return D.useEffect(()=>{const i=setTimeout(()=>{a(r)},t);return()=>{clearTimeout(i)}},[r,t]),n}const $b=[{id:"math-random",pattern:/\bMath\.random\s*\(\s*\)/g,severity:"error",message:"检测到 Math.random()",suggestion:"请使用 window.__motionUtils.seededRandom(time) 替代"},{id:"date-now",pattern:/\bDate\.now\s*\(\s*\)/g,severity:"warning",message:"检测到 Date.now()",suggestion:"请使用 render 函数的 time 参数替代"},{id:"new-date",pattern:/\bnew\s+Date\s*\(/g,severity:"warning",message:"检测到 new Date()",suggestion:"请使用 render 函数的 time 参数替代"},{id:"performance-now",pattern:/\bperformance\.now\s*\(\s*\)/g,severity:"warning",message:"检测到 performance.now()",suggestion:"请使用 render 函数的 time 参数替代"}];function Ub(r){const t=[];for(const n of $b){n.pattern.lastIndex=0;let a;for(;(a=n.pattern.exec(r))!==null;)t.push({severity:n.severity,message:n.message,match:a[0],suggestion:n.suggestion,position:a.index})}return t.sort((n,a)=>n.position-a.position),{isValid:!t.some(n=>n.severity==="error"),issues:t}}function zb({validation:r,onDismiss:t}){const{t:n}=ft();if(r.isValid&&r.issues.length===0)return null;const a=r.issues.filter(s=>s.severity==="error").length,i=r.issues.filter(s=>s.severity==="warning").length;return g.jsx("div",{className:"absolute top-2 left-2 right-2 z-10",children:g.jsx("div",{className:"bg-yellow-50 border border-yellow-200 rounded-lg p-3 shadow-sm",children:g.jsxs("div",{className:"flex items-start gap-2",children:[g.jsx("span",{className:"text-yellow-600 text-lg",children:(a>0,"!")}),g.jsxs("div",{className:"flex-1 min-w-0",children:[g.jsxs("h4",{className:"text-sm font-medium text-yellow-800",children:[n("codeWarning.title"),a>0&&g.jsx("span",{className:"ml-2 text-red-600",children:n("codeWarning.errorCount",{count:a})}),i>0&&g.jsx("span",{className:"ml-2 text-yellow-600",children:n("codeWarning.warningCount",{count:i})})]}),g.jsxs("ul",{className:"mt-1 text-xs text-yellow-700 space-y-1",children:[r.issues.slice(0,3).map((s,c)=>g.jsxs("li",{className:"flex items-start gap-1",children:[g.jsx("span",{className:s.severity==="error"?"text-red-500":"text-yellow-500",children:(s.severity==="error","!")}),g.jsxs("span",{children:[s.message,": ",g.jsx("code",{className:"bg-yellow-100 px-1 rounded",children:s.match})]})]},c)),r.issues.length>3&&g.jsx("li",{className:"text-yellow-600",children:n("codeWarning.moreIssues",{count:r.issues.length-3})})]}),g.jsx("p",{className:"mt-2 text-xs text-yellow-600",children:n("codeWarning.hint")})]}),t&&g.jsx("button",{onClick:t,className:"text-yellow-400 hover:text-yellow-600 transition-colors","aria-label":n("codeWarning.dismiss"),children:g.jsx("svg",{className:"w-4 h-4",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M6 18L18 6M6 6l12 12"})})})]})})})}function pm(r){const t=Math.floor(r/1e3),n=Math.floor(t/60),a=t%60;return`${n.toString().padStart(2,"0")}:${a.toString().padStart(2,"0")}`}function Hb(){const{t:r}=ft(),{currentMotion:t,isPlaying:n,currentTime:a,setIsPlaying:i,setCurrentTime:s,aspectRatio:c,setRenderError:u,clearErrorState:p,previewBackgroundUrl:f,setPreviewBackgroundUrl:h,toasts:x,addToast:y,removeToast:w}=Ht(),E=D.useRef(null),C=D.useRef(null),S=D.useRef(null),P=D.useRef(null),b=D.useRef(null),A=D.useRef(null),T=D.useRef(null),[B,F]=D.useState(!1),[k,N]=D.useState(!1),[I,L]=D.useState(0),[M,H]=D.useState(!0),z=D.useMemo(()=>t!=null&&t.code?Ub(t.code):null,[t==null?void 0:t.code]);D.useEffect(()=>{H(!0)},[t==null?void 0:t.code]);const Z=D.useCallback(ge=>{console.log("[PreviewCanvas] Received render error:",ge.message),u(ge),F(!0)},[u]),oe=D.useCallback(ge=>{console.log("[PreviewCanvas] 动态时长已更新:",ge),L(ge)},[]),re=D.useCallback(ge=>{console.log("[PreviewCanvas] 性能警告，单帧渲染耗时:",ge.elapsed.toFixed(0),"ms"),i(!1),y({type:"warning",message:r("preview.performanceWarning",{elapsed:(ge.elapsed/1e3).toFixed(1)}),duration:5e3})},[i,y]),[se,Y]=D.useState({width:640,height:360}),te=(t==null?void 0:t.id)??null,ee=(t==null?void 0:t.code)??null,_=D.useCallback(ge=>{C.current=ge,N(!!ge)},[]),$=D.useCallback(()=>{if(!S.current)return;const ge=S.current.getBoundingClientRect(),je=ge.width-32,Ye=ge.height-32;if(je>0&&Ye>0){const tt=gC(c,je,Ye);Y(tt)}},[c]);D.useEffect(()=>{$();const ge=()=>{$()};return window.addEventListener("resize",ge),()=>window.removeEventListener("resize",ge)},[$]);const G=D.useRef(null);D.useEffect(()=>{if(console.log("[PreviewCanvas] 检查渲染器初始化, motionId:",te,"上一个:",A.current,"containerMounted:",k),!C.current||!t){console.log("[PreviewCanvas] container 或 motion 为空，清理渲染器"),P.current&&(P.current.destroy(),P.current=null),A.current=null,T.current=null,G.current=null,F(!1);return}const ge=A.current!==te,je=T.current!==ee,Ye=!G.current||G.current.width!==se.width||G.current.height!==se.height;if(!ge&&!je&&!Ye){console.log("[PreviewCanvas] motion ID、code 和尺寸均未变化，跳过重建");return}console.log("[PreviewCanvas] 需要重建渲染器, idChanged:",ge,"codeChanged:",je,"dimsChanged:",Ye),P.current&&(console.log("[PreviewCanvas] 清理旧渲染器"),P.current.destroy()),C.current.innerHTML="";const tt={...t,width:se.width,height:se.height};p();let Dt=!1;const ke=ye=>{Dt=!0,Z(ye)};console.log("[PreviewCanvas] 创建新渲染器, renderMode:",t.renderMode,"previewDims:",se);const xe=Gg(tt,{onError:ke,onDurationChange:oe,onPerformanceWarning:re});try{if(xe.initialize(C.current,tt),P.current=xe,A.current=te,T.current=ee,G.current={...se},Dt)console.log("[PreviewCanvas] 渲染器初始化完成但有错误，不自动播放"),F(!0),i(!1),s(0);else{F(!1),console.log("[PreviewCanvas] 渲染器初始化成功");const ye=xe;if(ye.getDuration){const Fe=ye.getDuration();L(Fe),console.log("[PreviewCanvas] 初始动态时长:",Fe)}s(0),xe.play(),i(!0),console.log("[PreviewCanvas] 新动效自动播放已启动")}}catch(ye){console.error("[PreviewCanvas] 渲染器初始化失败:",ye),F(!0),i(!1),s(0)}},[te,ee,t,k,se,i,s,Z,oe,re,p]),D.useEffect(()=>()=>{P.current&&(console.log("[PreviewCanvas] 组件卸载，清理渲染器"),P.current.destroy(),P.current=null)},[]);const K=(t==null?void 0:t.updatedAt)??0,ue=Mb(K,200),me=D.useRef(!0),Le=D.useRef(0);D.useEffect(()=>{!P.current||!t||A.current===te&&(console.log("[PreviewCanvas] 参数已更新 (updatedAt:",K,")，同步到渲染器"),P.current.updateDuration&&P.current.updateDuration(t.duration),t.parameters.forEach(ge=>{var Ye;let je;switch(ge.type){case"number":je=ge.value;break;case"color":je=ge.colorValue;break;case"select":je=ge.selectedValue;break;case"boolean":je=ge.boolValue;break;case"image":je=ge.imageValue;break;case"video":je=ge.videoValue;break;case"string":je=ge.stringValue;break}je!==void 0&&(console.log("[PreviewCanvas] 同步参数:",ge.id,"=",je),(Ye=P.current)==null||Ye.updateParameter(ge.id,je))}))},[K,te,t]),D.useEffect(()=>{if(me.current){me.current=!1,Le.current=ue;return}if(Le.current!==ue){if(Le.current=ue,!P.current||!t||B){console.log("[PreviewCanvas] 跳过自动重播: 渲染器不可用或有错误");return}A.current===te&&(console.log("[PreviewCanvas] 参数变更后自动重播 (debouncedUpdatedAt:",ue,")"),P.current.seek(0),P.current.play(),i(!0),s(0))}},[ue,te,t,B,i,s]),D.useEffect(()=>{if(P.current)return n?(P.current.play(),b.current=window.setInterval(()=>{P.current&&s(P.current.getCurrentTime())},100)):(P.current.pause(),b.current&&(clearInterval(b.current),b.current=null)),()=>{b.current&&(clearInterval(b.current),b.current=null)}},[n,s]);const he=D.useCallback(()=>{i(!n)},[n,i]),Ve=D.useCallback(()=>{P.current&&P.current.stop(),i(!1),s(0)},[i,s]),ot=I||(t==null?void 0:t.duration)||0,Vt=Math.min(a,ot),We=D.useCallback(ge=>{var Dt;const je=(Dt=ge.target.files)==null?void 0:Dt[0];if(!je)return;if(!["image/png","image/jpeg","image/webp"].includes(je.type)){alert(r("preview.imageTypeError")),ge.target.value="";return}je.size>10*1024*1024&&console.warn("[PreviewCanvas] 背景图较大，建议使用小于 10MB 的图片");const tt=URL.createObjectURL(je);h(tt),ge.target.value=""},[h]),Ee=D.useCallback(()=>{h(null)},[h]);return g.jsxs("div",{className:"flex flex-col h-full",children:[g.jsx("input",{ref:E,type:"file",accept:"image/png,image/jpeg,image/webp",onChange:We,className:"hidden"}),g.jsx("div",{className:"p-3 border-b border-border-default bg-background-elevated shrink-0",children:g.jsx("h2",{className:"text-sm font-medium font-display text-text-primary",children:r("preview.title")})}),g.jsx("div",{ref:S,className:"flex-1 flex items-center justify-center bg-background-primary p-4 overflow-hidden relative",children:g.jsxs("div",{className:"relative flex items-center justify-center bg-gray-900/50 rounded-lg overflow-hidden",style:{width:se.width,height:se.height},children:[g.jsxs("div",{className:"absolute inset-0 pointer-events-none z-10",children:[g.jsx("div",{className:"absolute inset-0 border-2 border-dashed border-accent-primary/50 rounded-md shadow-neon-soft"}),g.jsx("div",{className:"absolute top-0 left-0 w-4 h-4 border-t-2 border-l-2 border-accent-primary rounded-tl-sm"}),g.jsx("div",{className:"absolute top-0 right-0 w-4 h-4 border-t-2 border-r-2 border-accent-primary rounded-tr-sm"}),g.jsx("div",{className:"absolute bottom-0 left-0 w-4 h-4 border-b-2 border-l-2 border-accent-primary rounded-bl-sm"}),g.jsx("div",{className:"absolute bottom-0 right-0 w-4 h-4 border-b-2 border-r-2 border-accent-primary rounded-br-sm"})]}),t?g.jsxs("div",{className:"relative w-full h-full",children:[z&&z.issues.length>0&&M&&g.jsx(zb,{validation:z,onDismiss:()=>H(!1)}),g.jsx("div",{className:"w-full h-full overflow-hidden rounded-md",style:{backgroundImage:f?`url(${f})`:"none",backgroundSize:"cover",backgroundPosition:"center",backgroundRepeat:"no-repeat"},children:g.jsx("div",{ref:_,className:"overflow-hidden",style:{width:"100%",height:"100%",backgroundColor:f?"transparent":"#0A0A0F"}})}),g.jsxs("div",{className:"absolute bottom-2 right-2 flex items-center gap-2",children:[f&&g.jsx(lt,{variant:"ghost",size:"sm",onClick:Ee,className:"text-xs opacity-70 hover:opacity-100",children:r("preview.clearBackground")}),g.jsx(lt,{variant:"ghost",size:"sm",onClick:()=>{var ge;return(ge=E.current)==null?void 0:ge.click()},className:"text-xs opacity-70 hover:opacity-100",children:r(f?"preview.changeBackground":"preview.uploadBackground")})]}),g.jsx("div",{className:"absolute bottom-2 left-2 text-xs text-text-muted opacity-60 font-body",children:r("preview.backgroundHint")})]}):g.jsx("div",{className:"w-full h-full flex items-center justify-center rounded-md",style:{backgroundColor:"#0A0A0F"},children:g.jsxs("div",{className:"text-center",children:[g.jsx("p",{className:"font-display text-text-primary/90",children:r("preview.area")}),g.jsx("p",{className:"text-sm mt-1 font-body text-text-secondary",children:r("preview.areaHint")}),g.jsx("p",{className:"text-xs mt-2 font-mono text-accent-primary",children:Vl(c).label})]})})]})}),g.jsxs("div",{className:"relative h-14 bg-background-elevated border-t border-border-default flex items-center justify-center gap-4 px-4 shrink-0 overflow-hidden",children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:Ve,disabled:!t,children:r("preview.reset")}),g.jsx(lt,{variant:"primary",size:"sm",onClick:he,disabled:!t,children:r(n?"preview.pause":"preview.play")}),g.jsxs("span",{className:"text-sm font-mono text-text-muted min-w-[100px] text-center",children:[pm(Vt)," / ",pm(ot)]})]}),g.jsx(sb,{toasts:x,onClose:w})]})}function Vb({parameter:r,onChange:t}){const n=r.value??r.min??0,a=r.min??0,i=r.max??100,s=r.step??1,c=u=>{const p=parseFloat(u.target.value);t(p)};return g.jsx("div",{className:"py-2",children:g.jsx(Lg,{label:r.name,min:a,max:i,step:s,value:n,onChange:c,showValue:!0})})}function Wb({parameter:r,onChange:t}){const n=r.colorValue??"#000000",a=i=>{t(i.target.value)};return g.jsx("div",{className:"py-2",children:g.jsx(Fg,{label:r.name,value:n,onChange:a})})}function Gb({parameter:r,onChange:t}){var s,c;const n=r.selectedValue??((c=(s=r.options)==null?void 0:s[0])==null?void 0:c.value)??"",a=r.options??[],i=u=>{t(u.target.value)};return g.jsx("div",{className:"py-2",children:g.jsx(kl,{label:r.name,options:a,value:n,onChange:i})})}function qb({parameter:r,onChange:t}){const n=r.boolValue??!1,a=i=>{t(i.target.checked)};return g.jsx("div",{className:"py-2",children:g.jsx(Od,{label:r.name,checked:n,onChange:a})})}function Kb({parameter:r,onChange:t}){const{t:n}=ft(),[a,i]=D.useState(!1),[s,c]=D.useState(null),[u,p]=D.useState(!1),f=D.useRef(null),h=D.useRef(null),x=r.imageValue??r.placeholderImage,y=Pg(x),w=r.imageFileName;D.useEffect(()=>()=>{h.current&&clearTimeout(h.current)},[]);const E=D.useCallback(F=>{h.current&&clearTimeout(h.current),c(F),h.current=setTimeout(()=>{c(null),h.current=null},3e3)},[]),C=D.useCallback(async F=>{i(!0),c(null);try{const k=await nC(F);t(k.blobUrl)}catch(k){const N=k instanceof Error?k.message:"READ_ERROR";E(N)}finally{i(!1)}},[t,E]),S=()=>{var F;(F=f.current)==null||F.click()},P=F=>{var N;const k=(N=F.target.files)==null?void 0:N[0];k&&C(k),F.target.value=""},b=F=>{F.preventDefault(),F.stopPropagation(),p(!0)},A=F=>{F.preventDefault(),F.stopPropagation(),p(!1)},T=F=>{var N;F.preventDefault(),F.stopPropagation(),p(!1);const k=(N=F.dataTransfer.files)==null?void 0:N[0];k&&C(k)},B=y?Al(48,48):x;return g.jsxs("div",{className:"py-2",children:[g.jsx("label",{className:"block text-sm font-medium text-slate-300 mb-1.5",children:r.name}),g.jsxs("div",{onClick:S,onDragOver:b,onDragLeave:A,onDrop:T,className:`
          relative flex items-center gap-3 p-3 rounded-lg border-2 border-dashed cursor-pointer
          transition-colors duration-200
          ${u?"border-blue-400 bg-blue-500/20 shadow-[0_0_20px_rgba(59,130,246,0.15)]":"border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10"}
          ${a?"opacity-60 pointer-events-none":""}
        `,children:[g.jsx("div",{className:"flex-shrink-0 w-12 h-12 rounded overflow-hidden bg-slate-700/50 ring-1 ring-white/10",children:B&&g.jsx("img",{src:B,alt:r.name,className:"w-full h-full object-cover"})}),g.jsx("div",{className:"flex-1 min-w-0",children:a?g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx("div",{className:"w-4 h-4 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"}),g.jsx("span",{className:"text-sm text-slate-400",children:"处理中..."})]}):w?g.jsxs("div",{children:[g.jsx("p",{className:"text-sm text-slate-200 truncate",children:w}),g.jsx("p",{className:"text-xs text-slate-500",children:"点击或拖拽更换图片"})]}):g.jsxs("div",{children:[g.jsx("p",{className:"text-sm text-slate-300",children:"点击或拖拽上传图片"}),g.jsx("p",{className:"text-xs text-slate-500",children:"支持 PNG、JPEG 格式"})]})}),g.jsx("input",{ref:f,type:"file",accept:na.ACCEPTED_EXTENSIONS.join(","),onChange:P,className:"hidden"})]}),s&&g.jsxs("div",{className:"mt-2 flex items-start gap-2 p-2 bg-red-500/10 border border-red-500/30 rounded-md",children:[g.jsx("svg",{className:"flex-shrink-0 w-4 h-4 mt-0.5 text-red-400",fill:"currentColor",viewBox:"0 0 20 20",children:g.jsx("path",{fillRule:"evenodd",d:"M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z",clipRule:"evenodd"})}),g.jsx("p",{className:"text-xs text-red-300",children:n(w2[s]||"error.upload.fallback")})]})]})}function Xb({parameter:r,onChange:t}){const{t:n}=ft(),[a,i]=D.useState(!1),[s,c]=D.useState(null),[u,p]=D.useState(!1),f=D.useRef(null),h=D.useRef(null),x=D.useRef(null),y=r.videoValue??r.placeholderVideo,w=kg(y),E=r.videoFileName;D.useEffect(()=>()=>{x.current&&clearTimeout(x.current)},[]);const C=D.useCallback(N=>{x.current&&clearTimeout(x.current),c(N),x.current=setTimeout(()=>{c(null),x.current=null},3e3)},[]),S=D.useCallback(async N=>{i(!0),c(null);try{const I=await pC(N);t(I.blobUrl,I)}catch(I){const L=I instanceof Error?I.message:"LOAD_ERROR";C(L)}finally{i(!1)}},[t,C]),P=()=>{var N;(N=f.current)==null||N.click()},b=N=>{var L;const I=(L=N.target.files)==null?void 0:L[0];I&&S(I),N.target.value=""},A=N=>{N.preventDefault(),N.stopPropagation(),p(!0)},T=N=>{N.preventDefault(),N.stopPropagation(),p(!1)},B=N=>{var L;N.preventDefault(),N.stopPropagation(),p(!1);const I=(L=N.dataTransfer.files)==null?void 0:L[0];I&&S(I)},F=N=>{if(!N)return"";const I=Math.round(N/1e3),L=Math.floor(I/60),M=I%60;return L>0?`${L}:${M.toString().padStart(2,"0")}`:`${M}秒`},k=w?Pl(48,48):y;return g.jsxs("div",{className:"py-2",children:[g.jsx("label",{className:"block text-sm font-medium text-slate-300 mb-1.5",children:r.name}),g.jsxs("div",{onClick:P,onDragOver:A,onDragLeave:T,onDrop:B,className:`
          relative flex items-center gap-3 p-3 rounded-lg border-2 border-dashed cursor-pointer
          transition-colors duration-200
          ${u?"border-blue-400 bg-blue-500/20 shadow-[0_0_20px_rgba(59,130,246,0.15)]":"border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10"}
          ${a?"opacity-60 pointer-events-none":""}
        `,children:[g.jsx("div",{className:"flex-shrink-0 w-12 h-12 rounded overflow-hidden bg-slate-700/50 ring-1 ring-white/10",children:w?g.jsx("img",{src:k,alt:r.name,className:"w-full h-full object-cover"}):g.jsx("video",{ref:h,src:y,className:"w-full h-full object-cover",muted:!0,playsInline:!0,onLoadedData:N=>{const I=N.currentTarget;I.currentTime=0}})}),g.jsx("div",{className:"flex-1 min-w-0",children:a?g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx("div",{className:"w-4 h-4 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"}),g.jsx("span",{className:"text-sm text-slate-400",children:"处理中..."})]}):E?g.jsxs("div",{children:[g.jsx("p",{className:"text-sm text-slate-200 truncate",children:E}),g.jsxs("p",{className:"text-xs text-slate-500",children:[r.videoDuration&&F(r.videoDuration),r.videoDuration&&" · ","点击或拖拽更换视频"]})]}):g.jsxs("div",{children:[g.jsx("p",{className:"text-sm text-slate-300",children:"点击或拖拽上传视频"}),g.jsx("p",{className:"text-xs text-slate-500",children:"支持 MP4、WebM 格式"})]})}),g.jsx("input",{ref:f,type:"file",accept:Ol.ACCEPTED_EXTENSIONS.join(","),onChange:b,className:"hidden"})]}),s&&g.jsxs("div",{className:"mt-2 flex items-start gap-2 p-2 bg-red-500/10 border border-red-500/30 rounded-md",children:[g.jsx("svg",{className:"flex-shrink-0 w-4 h-4 mt-0.5 text-red-400",fill:"currentColor",viewBox:"0 0 20 20",children:g.jsx("path",{fillRule:"evenodd",d:"M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z",clipRule:"evenodd"})}),g.jsx("p",{className:"text-xs text-red-300",children:n(E2[s]||"error.upload.fallback")})]})]})}const Yb=300;function Qb({parameter:r,onChange:t}){const[n,a]=D.useState(r.stringValue??""),i=D.useRef(null);D.useEffect(()=>{a(r.stringValue??"")},[r.stringValue,r.id]);const s=D.useCallback(c=>{const u=c.target.value;a(u),i.current&&clearTimeout(i.current),i.current=setTimeout(()=>{t(u)},Yb)},[t]);return g.jsxs("div",{className:"py-2",children:[g.jsx("label",{className:"block text-sm text-[var(--color-text-secondary)] mb-1",children:r.name}),g.jsx("input",{type:"text",value:n,onChange:s,placeholder:r.placeholder??"",maxLength:r.maxLength,className:"w-full px-3 py-2 bg-[var(--color-bg-tertiary)] border border-[var(--color-border)] rounded text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-primary)] overflow-x-auto"})]})}function Jb(){const{t:r}=ft(),{currentMotion:t,updateMotionParameter:n,updateVideoParameter:a,touchConversation:i,saveCurrentConversation:s}=Ht(),c=D.useCallback((f,h)=>{t&&(console.log("[ParameterPanel] 参数变更:",f,h),n(f,h),i(),s())},[t,n,i,s]),u=D.useCallback((f,h,x)=>{t&&(console.log("[ParameterPanel] 视频参数变更:",f,x),a(f,h,x),i(),s())},[t,a,i,s]),p=f=>{switch(f.type){case"number":return g.jsx(Vb,{parameter:f,onChange:h=>c(f.id,h)},f.id);case"color":return g.jsx(Wb,{parameter:f,onChange:h=>c(f.id,h)},f.id);case"select":return g.jsx(Gb,{parameter:f,onChange:h=>c(f.id,h)},f.id);case"boolean":return g.jsx(qb,{parameter:f,onChange:h=>c(f.id,h)},f.id);case"image":return g.jsx(Kb,{parameter:f,onChange:h=>c(f.id,h)},f.id);case"video":return g.jsx(Xb,{parameter:f,onChange:(h,x)=>u(f.id,h,x)},f.id);case"string":return g.jsx(Qb,{parameter:f,onChange:h=>c(f.id,h)},f.id);default:return null}};return!t||t.parameters.length===0?g.jsx("div",{className:"text-center font-body text-text-muted py-8",children:r("panel.emptyParameters")}):g.jsx("div",{className:"space-y-1 divide-y divide-border-default/50",children:t.parameters.map(p)})}function Zb({question:r,progress:t,onSelectOption:n,onSubmitCustom:a,onSkip:i,disabled:s=!1}){const{t:c}=ft(),[u,p]=D.useState(!1),[f,h]=D.useState(""),x=C=>{s||(p(!1),h(""),n(C))},y=()=>{s||p(!0)},w=()=>{s||!f.trim()||(a(f.trim()),h(""),p(!1))},E=C=>{C.key==="Enter"&&!C.shiftKey&&(C.preventDefault(),w())};return!r||!r.options||r.options.length===0?g.jsxs("div",{className:"text-sm text-[var(--color-text-secondary)]",children:[c("clarify.loadFailed"),g.jsx("button",{onClick:i,className:"ml-2 text-[var(--color-primary)] hover:underline",children:c("common.skip")})]}):g.jsxs("div",{className:"space-y-3",children:[g.jsxs("div",{className:"flex items-center justify-between text-xs text-[var(--color-text-secondary)]",children:[g.jsx("span",{children:c("clarify.progress",{current:t.current,total:t.total})}),g.jsx("button",{onClick:i,disabled:s,className:"text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors disabled:opacity-50",children:c("clarify.skipGenerate")})]}),g.jsx("div",{className:"bg-[var(--color-surface-elevated)] rounded-[var(--border-radius)] px-3 py-2",children:g.jsx("p",{className:"text-sm text-[var(--color-text-primary)]",children:r.question})}),g.jsxs("div",{className:"flex flex-wrap gap-2",children:[r.options.map(C=>g.jsxs("button",{onClick:()=>x(C.id),disabled:s,className:`
              px-3 py-1.5 text-sm rounded-[var(--border-radius)]
              border border-[var(--color-border)]
              bg-[var(--color-surface)] text-[var(--color-text-primary)]
              hover:bg-[var(--color-primary)] hover:text-white hover:border-[var(--color-primary)]
              transition-colors
              disabled:opacity-50 disabled:cursor-not-allowed
            `,children:[g.jsxs("span",{className:"font-medium mr-1",children:[C.id,"."]}),C.label]},C.id)),g.jsx("button",{onClick:y,disabled:s,className:`
            px-3 py-1.5 text-sm rounded-[var(--border-radius)]
            border border-dashed border-[var(--color-border)]
            bg-transparent text-[var(--color-text-secondary)]
            hover:border-[var(--color-primary)] hover:text-[var(--color-primary)]
            transition-colors
            disabled:opacity-50 disabled:cursor-not-allowed
            ${u?"border-[var(--color-primary)] text-[var(--color-primary)]":""}
          `,children:c("common.custom")})]}),u&&g.jsxs("div",{className:"flex gap-2",children:[g.jsx(wo,{value:f,onChange:C=>h(C.target.value),onKeyDown:E,placeholder:c("clarify.customPlaceholder"),disabled:s,className:"flex-1",autoFocus:!0}),g.jsx(lt,{variant:"primary",size:"sm",onClick:w,disabled:s||!f.trim(),children:c("common.confirm")}),g.jsx(lt,{variant:"ghost",size:"sm",onClick:()=>{p(!1),h("")},disabled:s,children:c("common.cancel")})]})]})}function e4({error:r,onFix:t,loading:n=!1,disabled:a=!1,attemptCount:i=0,maxAttempts:s=Fs.MAX_FIX_ATTEMPTS}){const{t:c}=ft(),u=i>=s;return g.jsxs("div",{className:"bg-[var(--color-warning)]/10 border border-[var(--color-warning)]/30 rounded-[var(--border-radius)] px-3 py-3",children:[g.jsxs("div",{className:"flex items-start gap-2",children:[g.jsx("span",{className:"text-[var(--color-warning)] text-lg flex-shrink-0",children:"⚠️"}),g.jsxs("div",{className:"flex-1 min-w-0",children:[g.jsx("p",{className:"text-sm text-[var(--color-text-primary)] font-medium",children:c("error.codeExecution")}),g.jsx("p",{className:"text-sm text-[var(--color-text-secondary)] mt-1",children:c(r.friendlyMessage)})]})]}),i>0&&!u&&g.jsx("div",{className:"mt-2 text-xs text-[var(--color-text-secondary)]",children:c("error.retryCount",{count:i,remaining:s-i})}),g.jsx("div",{className:"mt-3",children:u?g.jsxs("div",{className:"text-sm text-[var(--color-text-secondary)] bg-[var(--color-surface)] rounded-[var(--border-radius-sm)] p-3",children:[g.jsx("p",{className:"font-medium text-[var(--color-text-primary)] mb-2",children:c("error.maxRetryTitle",{max:s})}),g.jsx("p",{className:"mb-1",children:c("error.maxRetrySuggestion")}),g.jsxs("ol",{className:"list-decimal list-inside space-y-1 ml-1",children:[g.jsx("li",{children:c("error.maxRetry1")}),g.jsx("li",{children:c("error.maxRetry2")}),g.jsx("li",{children:c("error.maxRetry3")})]}),g.jsx("p",{className:"mt-2",children:c("error.maxRetryHint")})]}):g.jsx(lt,{variant:"primary",size:"sm",onClick:t,loading:n,disabled:a||n,children:c(n?"error.fixing":"error.autoFix")})})]})}function t4({onFileSelect:r,disabled:t=!1,currentCount:n=0}){const a=D.useRef(null),i=n>=Ir.MAX_ATTACHMENTS,s=D.useCallback(()=>{!t&&!i&&a.current&&a.current.click()},[t,i]),c=D.useCallback(p=>{const f=p.target.files;f&&f.length>0&&r(f),a.current&&(a.current.value="")},[r]),u=[...Ir.ACCEPTED_IMAGE_FORMATS,...Ir.ACCEPTED_DOCUMENT_EXTENSIONS].join(",");return g.jsxs(g.Fragment,{children:[g.jsx("input",{ref:a,type:"file",accept:u,multiple:!0,onChange:c,className:"hidden","aria-hidden":"true"}),g.jsx("button",{type:"button",onClick:s,disabled:t||i,title:i?"最多支持 5 个附件":"添加附件",className:`
          flex items-center justify-center
          w-8 h-8 rounded-[var(--border-radius)]
          transition-colors duration-150
          ${t||i?"text-[var(--color-text-secondary)] opacity-50 cursor-not-allowed":"text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-elevated)]"}
        `,children:g.jsx("svg",{width:"20",height:"20",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:g.jsx("path",{d:"M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"})})})]})}function r4({attachment:r,onRemove:t,disabled:n=!1}){var w;const{t:a}=ft(),{tempId:i,file:s,status:c,previewUrl:u,error:p,wasCompressed:f}=r,h=s.type.startsWith("image/"),x=c==="pending"||c==="processing",y=c==="error";return g.jsxs("div",{className:`
        relative group
        w-20 h-20 rounded-lg
        overflow-hidden
        border border-border-default
        ${y?"border-accent-tertiary":""}
        ${x?"animate-pulse":""}
      `,children:[h&&u&&g.jsx("img",{src:u,alt:s.name,className:"w-full h-full object-cover"}),!h&&g.jsxs("div",{className:"w-full h-full flex flex-col items-center justify-center bg-background-elevated p-1",children:[g.jsxs("svg",{width:"24",height:"24",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",className:"text-text-muted",children:[g.jsx("path",{d:"M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"}),g.jsx("polyline",{points:"14,2 14,8 20,8"}),g.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),g.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"}),g.jsx("line",{x1:"10",y1:"9",x2:"8",y2:"9"})]}),g.jsx("span",{className:"text-[10px] font-body text-text-muted mt-1 truncate max-w-full px-1",children:(w=s.name.split(".").pop())==null?void 0:w.toUpperCase()})]}),x&&g.jsx("div",{className:"absolute inset-0 bg-black/40 flex items-center justify-center",children:g.jsx("div",{className:"w-5 h-5 border-2 border-accent-primary border-t-transparent rounded-full animate-spin"})}),y&&g.jsx("div",{className:"absolute inset-0 bg-accent-tertiary/20 flex items-center justify-center",children:g.jsxs("svg",{width:"20",height:"20",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",className:"text-accent-tertiary",children:[g.jsx("circle",{cx:"12",cy:"12",r:"10"}),g.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),g.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),!n&&g.jsx("button",{type:"button",onClick:()=>t(i),className:`
            absolute top-1 right-1
            w-5 h-5 rounded-full
            bg-black/60 hover:bg-black/80
            flex items-center justify-center
            opacity-0 group-hover:opacity-100
            transition-opacity duration-150
            cursor-pointer
          `,title:"移除",children:g.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"white",strokeWidth:"2",children:[g.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),g.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})}),f&&c==="ready"&&g.jsx("div",{className:"absolute bottom-0 left-0 right-0 bg-accent-secondary/80 text-white text-[9px] text-center py-0.5",title:"图片已压缩以优化传输",children:"已压缩"}),g.jsx("div",{className:"absolute bottom-0 left-0 right-0 bg-black/60 text-white text-[9px] font-body truncate px-1 py-0.5 opacity-0 group-hover:opacity-100 transition-opacity",title:s.name,children:s.name}),y&&p&&g.jsx("div",{className:"absolute -bottom-6 left-0 right-0 text-accent-tertiary text-[10px] font-body truncate",title:a(p),children:a(p)})]})}function n4({attachments:r,onRemove:t,disabled:n=!1}){if(r.length===0)return null;const a=r.length>=Ir.MAX_ATTACHMENTS;return g.jsxs("div",{className:"px-3 py-2 border-b border-[var(--color-border)]",children:[g.jsx("div",{className:"flex flex-wrap gap-2",children:r.map(i=>g.jsx(r4,{attachment:i,onRemove:t,disabled:n},i.tempId))}),g.jsxs("div",{className:"mt-2 flex items-center justify-between text-[11px] text-[var(--color-text-secondary)]",children:[g.jsxs("span",{children:[r.length," / ",Ir.MAX_ATTACHMENTS," 个附件"]}),a&&g.jsx("span",{className:"text-[var(--color-warning)]",children:"已达上限"})]})]})}function o4({messageId:r,attachmentIds:t}){const[n,a]=D.useState([]),[i,s]=D.useState(!0),[c,u]=D.useState(null),{loadMessageAttachments:p}=Ht();D.useEffect(()=>{let x=!0;async function y(){try{const w=await p(r);if(x){const E=t.map(C=>w.find(S=>S.id===C)).filter(C=>C!==void 0);a(E)}}catch(w){console.error("[MessageAttachment] 加载附件失败:",w)}finally{x&&s(!1)}}return y(),()=>{x=!1}},[r,t,p]);const f=D.useCallback(x=>{u(x)},[]),h=D.useCallback(()=>{u(null)},[]);return i?g.jsx("div",{className:"flex gap-1 mt-1",children:t.map(x=>g.jsx("div",{className:"w-12 h-12 rounded bg-black/20 animate-pulse"},x))}):n.length===0?null:g.jsxs(g.Fragment,{children:[g.jsx("div",{className:"flex flex-wrap gap-1 mt-1",children:n.map(x=>g.jsx(a4,{attachment:x,onImageClick:f},x.id))}),c&&g.jsx("div",{className:"fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4",onClick:h,children:g.jsxs("div",{className:"relative max-w-full max-h-full",children:[g.jsx("img",{src:c,alt:"预览大图",className:"max-w-full max-h-[90vh] object-contain",onClick:x=>x.stopPropagation()}),g.jsx("button",{onClick:h,className:"absolute top-2 right-2 w-8 h-8 bg-black/60 hover:bg-black/80 rounded-full flex items-center justify-center text-white",title:"关闭",children:g.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:[g.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),g.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})})]})})]})}function a4({attachment:r,onImageClick:t}){var a;return r.type==="image"?g.jsx("button",{type:"button",onClick:()=>t(r.content),className:"w-12 h-12 rounded overflow-hidden border border-white/20 hover:border-white/40 transition-colors cursor-pointer",title:`${r.fileName}${r.wasCompressed?" (已压缩)":""}`,children:g.jsx("img",{src:r.content,alt:r.fileName,className:"w-full h-full object-cover"})}):g.jsxs("div",{className:"w-12 h-12 rounded overflow-hidden border border-white/20 bg-black/20 flex flex-col items-center justify-center p-1",title:r.fileName,children:[g.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",className:"text-white/70",children:[g.jsx("path",{d:"M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"}),g.jsx("polyline",{points:"14,2 14,8 20,8"}),g.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),g.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),g.jsx("span",{className:"text-[8px] text-white/60 mt-0.5 truncate max-w-full",children:(a=r.fileName.split(".").pop())==null?void 0:a.toUpperCase()})]})}const i4=50;function s4(r){const t=new Date,n=new Date(r),a=Ct.t.bind(Ct),i=Ct.language==="en"?"en-US":"zh-CN",s=new Date(t.getFullYear(),t.getMonth(),t.getDate());if(n>=s)return n.toLocaleTimeString(i,{hour:"2-digit",minute:"2-digit"});const c=new Date(s);if(c.setDate(c.getDate()-1),n>=c)return a("time.yesterday");const u=new Date(s);return u.setDate(u.getDate()-s.getDay()),n>=u?[a("time.sunday"),a("time.monday"),a("time.tuesday"),a("time.wednesday"),a("time.thursday"),a("time.friday"),a("time.saturday")][n.getDay()]:n.toLocaleDateString(i,{year:"numeric",month:"2-digit",day:"2-digit"})}function l4({conversation:r,isActive:t,disabled:n,isEditing:a,isSelectionMode:i=!1,isSelected:s=!1,onSelect:c,onDelete:u,onDuplicate:p,onExport:f,onStartEdit:h,onSaveEdit:x,onCancelEdit:y,onToggleSelect:w}){const{t:E}=ft(),[C,S]=D.useState(r.title),P=D.useRef(null);D.useEffect(()=>{a&&P.current&&(P.current.focus(),P.current.select())},[a]),D.useEffect(()=>{S(r.title)},[r.title]);const b=()=>{if(i){w==null||w();return}!n&&!a&&c()},A=I=>{I.stopPropagation(),n||u()},T=I=>{I.stopPropagation(),h()},B=I=>{I.stopPropagation(),n||p()},F=I=>{I.stopPropagation(),f()},k=I=>{I.key==="Enter"?(I.preventDefault(),x(C)):I.key==="Escape"&&(I.preventDefault(),S(r.title),y())},N=()=>{x(C)};return g.jsxs("div",{onClick:b,className:`
        group flex items-center justify-between px-3 py-2 cursor-pointer transition-colors
        ${t&&!i?"bg-[var(--color-surface-elevated)] border-l-2 border-[var(--color-primary)]":"hover:bg-[var(--color-surface-elevated)]"}
        ${s?"bg-[var(--color-primary)] bg-opacity-10":""}
        ${n&&!i?"opacity-50 cursor-not-allowed":""}
      `,title:i?E("history.clickToSelect"):n?E("history.processing"):r.title,children:[i&&g.jsx("div",{className:"mr-2 flex-shrink-0",children:g.jsx("div",{className:`
              w-4 h-4 rounded border-2 flex items-center justify-center transition-colors
              ${s?"bg-[var(--color-primary)] border-[var(--color-primary)]":"border-[var(--color-text-secondary)] hover:border-[var(--color-primary)]"}
            `,children:s&&g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-3 w-3 text-white",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:3,d:"M5 13l4 4L19 7"})})})}),g.jsxs("div",{className:"flex-1 min-w-0",children:[a?g.jsx("input",{ref:P,type:"text",value:C,onChange:I=>S(I.target.value),onKeyDown:k,onBlur:N,maxLength:i4,className:`
              w-full text-sm px-1 py-0.5 rounded
              bg-[var(--color-surface)] border border-[var(--color-primary)]
              text-[var(--color-text-primary)]
              focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)]
            `,onClick:I=>I.stopPropagation()}):g.jsx("div",{className:`
              text-sm truncate
              ${t?"text-[var(--color-text-primary)] font-medium":"text-[var(--color-text-primary)]"}
            `,children:r.title}),g.jsx("div",{className:"text-xs text-[var(--color-text-secondary)] mt-0.5",children:s4(r.updatedAt)})]}),!n&&!a&&!i&&g.jsxs("div",{className:"flex items-center opacity-0 group-hover:opacity-100 transition-all",children:[g.jsx("button",{onClick:T,className:`
              ml-1 p-1 rounded
              text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]
              hover:bg-[var(--color-primary)] hover:bg-opacity-10
              transition-all
            `,title:E("history.editTitle"),children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"})})}),g.jsx("button",{onClick:B,className:`
              ml-1 p-1 rounded
              text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]
              hover:bg-[var(--color-primary)] hover:bg-opacity-10
              transition-all
            `,title:E("history.duplicateConversation"),children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"})})}),g.jsx("button",{onClick:F,className:`
              ml-1 p-1 rounded
              text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]
              hover:bg-[var(--color-primary)] hover:bg-opacity-10
              transition-all
            `,title:E("history.exportConversation"),children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"})})}),g.jsx("button",{onClick:A,className:`
              ml-1 p-1 rounded
              text-[var(--color-text-secondary)] hover:text-[var(--color-error)]
              hover:bg-[var(--color-error)] hover:bg-opacity-10
              transition-all
            `,title:E("history.deleteConversation"),children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"})})})]})]})}function c4({isOpen:r,result:t,onClose:n}){const{t:a}=ft();if(!r||!t)return null;const i=t.errors.length>0,s=t.warnings&&t.warnings.length>0,c=t.success&&t.skippedCount>0;return g.jsxs("div",{className:"fixed inset-0 z-50 flex items-center justify-center",children:[g.jsx("div",{className:"absolute inset-0 bg-black bg-opacity-50",onClick:n}),g.jsxs("div",{className:"relative bg-[var(--color-surface)] rounded-lg shadow-xl max-w-md w-full mx-4 overflow-hidden",children:[g.jsx("div",{className:"px-4 py-3 border-b border-[var(--color-border)]",children:g.jsx("h3",{className:"text-lg font-medium text-[var(--color-text-primary)]",children:a("import.title")})}),g.jsxs("div",{className:"px-4 py-4",children:[g.jsxs("div",{className:"flex items-center gap-2 mb-3",children:[g.jsx("div",{className:`
                w-6 h-6 rounded-full flex items-center justify-center
                ${t.success?c?"bg-yellow-100 text-yellow-600":"bg-green-100 text-green-600":"bg-red-100 text-red-600"}
              `,children:t.success?c?g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"})}):g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M5 13l4 4L19 7"})}):g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-4 w-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M6 18L18 6M6 6l12 12"})})}),g.jsx("span",{className:"text-[var(--color-text-primary)]",children:t.success?a("import.successCount",{count:t.importedCount}):a("import.failed")})]}),t.skippedCount>0&&g.jsx("div",{className:"text-sm text-[var(--color-text-secondary)] mb-3",children:a("import.skippedCount",{count:t.skippedCount})}),s&&g.jsxs("div",{className:"mt-3 border-t border-[var(--color-border)] pt-3",children:[g.jsx("div",{className:"text-sm font-medium text-[var(--color-text-secondary)] mb-2",children:a("import.warnings")}),g.jsx("div",{className:"space-y-1",children:t.warnings.map((u,p)=>g.jsx("div",{className:"text-sm text-amber-900 dark:text-amber-100 bg-amber-100 dark:bg-amber-800/40 px-2 py-1 rounded",children:u},p))})]}),i&&g.jsxs("div",{className:"mt-3 border-t border-[var(--color-border)] pt-3",children:[g.jsx("div",{className:"text-sm font-medium text-[var(--color-text-secondary)] mb-2",children:a("import.errorDetails")}),g.jsx("div",{className:"max-h-32 overflow-y-auto space-y-1",children:t.errors.map((u,p)=>g.jsxs("div",{className:"text-sm text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/20 px-2 py-1 rounded",children:[u.title&&g.jsxs("span",{className:"font-medium",children:[u.title,": "]}),u.message]},p))})]})]}),g.jsx("div",{className:"px-4 py-3 border-t border-[var(--color-border)] flex justify-end",children:g.jsx("button",{onClick:n,className:"px-4 py-2 text-sm font-medium text-white bg-[var(--color-primary)] rounded hover:bg-opacity-90 transition-colors",children:a("common.ok")})})]})]})}function qg(r,t,n="application/json"){const a=new Blob([r],{type:n}),i=URL.createObjectURL(a),s=document.createElement("a");s.href=i,s.download=t,document.body.appendChild(s),s.click(),document.body.removeChild(s),URL.revokeObjectURL(i)}function u4(r){return new Promise((t,n)=>{const a=new FileReader;a.onload=()=>t(a.result),a.onerror=()=>n(new Error("读取文件失败")),a.readAsText(r)})}async function d4(r){if(r.startsWith("data:")||r.startsWith("__"))return r;const n=await(await fetch(r)).blob();return new Promise((a,i)=>{const s=new FileReader;s.onloadend=()=>a(s.result),s.onerror=()=>i(new Error("图片转换失败")),s.readAsDataURL(n)})}function f4(r){if(r.startsWith("__")||r.startsWith("blob:"))return r;const t=r.match(/^data:([^;]+);base64,(.+)$/);if(!t)return r;const n=t[1],a=t[2],i=atob(a),s=new ArrayBuffer(i.length),c=new Uint8Array(s);for(let p=0;p<i.length;p++)c[p]=i.charCodeAt(p);const u=new Blob([c],{type:n});return URL.createObjectURL(u)}function p4(r){return/^data:image\/[a-zA-Z]+;base64,/.test(r)}const Dl="0.1.0",Hd=".neon",Kg="application/json",Xg={VIDEO:"__VIDEO_PLACEHOLDER__"},h4="（导入）",lr={PARSE_ERROR:"文件格式错误，请确认是有效的 .neon 文件",VERSION_INCOMPATIBLE:"文件版本不兼容，请使用最新版本的应用",INVALID_FORMAT:"文件数据不完整，无法导入",DATA_CORRUPTED:"文件数据损坏，部分对话无法导入",STORAGE_FULL:"存储空间不足，请删除一些历史对话后重试"};async function m4(r){const t={...r};switch(r.type){case"image":if(r.imageValue&&r.imageValue.startsWith("blob:"))try{t.imageValue=await d4(r.imageValue)}catch(n){console.warn(`[Exporter] 图片序列化失败: ${r.id}`,n)}break;case"video":r.videoValue&&r.videoValue.startsWith("blob:")&&(t.videoValue=Xg.VIDEO);break}return t}async function g4(r){const t=await Promise.all(r.parameters.map(m4));return{...r,parameters:t}}async function Yg(r){let t=null;return r.motion&&(t=await g4(r.motion)),{id:r.id,title:r.title,messages:r.messages,motion:t,createdAt:r.createdAt,updatedAt:r.updatedAt}}async function x4(r,t){const n=await Yg(r),a={version:Dl,exportedAt:Date.now(),conversation:n},i=JSON.stringify(a,null,2),c=`${Qg(r.title)}${Hd}`;qg(i,c,Kg)}async function v4(r,t){const n=await Promise.all(r.map(async u=>{const p=await Yg(u);return{version:Dl,exportedAt:Date.now(),conversation:p}})),a=JSON.stringify(n,null,2),i=`conversations-export-${r.length}`,c=`${Qg(i)}${Hd}`;qg(a,c,Kg)}function Qg(r){return r.replace(/[<>:"/\\|?*]/g,"_").replace(/[\x00-\x1f]/g,"").replace(/\s+/g,"_").substring(0,100)}function y4(r){let t;try{t=JSON.parse(r)}catch{return{valid:!1,error:"PARSE_ERROR",errorMessage:lr.PARSE_ERROR}}const n=Array.isArray(t)?t:[t];if(n.length===0)return{valid:!1,error:"INVALID_FORMAT",errorMessage:lr.INVALID_FORMAT};const a=[],i=[];for(let s=0;s<n.length;s++){const c=n[s],u=w4(c,s);if(!u.valid)return u;u.warnings&&i.push(...u.warnings),a.push(c)}return{valid:!0,items:a,warnings:i.length>0?i:void 0}}function w4(r,t){const n=[],a=r.version;if(typeof a!="string")return{valid:!1,error:"INVALID_FORMAT",errorMessage:`${lr.INVALID_FORMAT}（项 ${t+1}：缺少版本号）`};if(a!==Dl&&n.push(`文件版本 ${a} 与当前版本 ${Dl} 不一致，部分功能可能无法正常工作`),typeof r.exportedAt!="number"||r.exportedAt<=0)return{valid:!1,error:"INVALID_FORMAT",errorMessage:`${lr.INVALID_FORMAT}（项 ${t+1}：缺少导出时间）`};if(!r.conversation||typeof r.conversation!="object")return{valid:!1,error:"INVALID_FORMAT",errorMessage:`${lr.INVALID_FORMAT}（项 ${t+1}：缺少对话数据）`};const i=E4(r.conversation,t);return i.valid?{valid:!0,warnings:n.length>0?n:void 0}:i}function E4(r,t=0){if(typeof r.id!="string"||r.id.length===0)return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：缺少对话 ID）`};if(typeof r.title!="string"||r.title.length===0)return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：缺少对话标题）`};if(!Array.isArray(r.messages))return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：消息列表无效）`};if(r.motion!==null&&typeof r.motion!="object")return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：动效数据无效）`};if(r.motion&&typeof r.motion=="object"){const n=r.motion;if(typeof n.code!="string")return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：动效代码无效）`};if(!Array.isArray(n.parameters))return{valid:!1,error:"DATA_CORRUPTED",errorMessage:`${lr.DATA_CORRUPTED}（项 ${t+1}：动效参数无效）`}}return{valid:!0}}const hm=pi;function C4(r){const t={...r};switch(r.type){case"image":r.imageValue&&p4(r.imageValue)&&(t.imageValue=f4(r.imageValue));break;case"video":r.videoValue===Xg.VIDEO&&(t.videoValue=void 0,t.videoFileName=void 0,t.videoDuration=void 0);break}return t}function b4(r){const t=r.parameters.map(C4);return{...r,parameters:t}}function S4(r){const t=Date.now();let n=null;return r.motion&&(n=b4(r.motion),n={...n,id:hm(),createdAt:t,updatedAt:t}),{id:hm(),title:r.title,messages:r.messages,motion:n,createdAt:t,updatedAt:t}}function R4(r,t){if(!t.includes(r))return r;const n=h4;let a=`${r}${n}`;if(!t.includes(a))return a;let i=2;for(;t.includes(a);)if(a=`${r}${n.slice(0,-1)} ${i}）`,i++,i>100){a=`${r}${n.slice(0,-1)} ${Date.now()}）`;break}return a}async function Jg(r,t){var p;const n={success:!1,importedCount:0,skippedCount:0,importedIds:[],errors:[]};let a;try{a=await u4(r)}catch{return n.errors.push({type:"PARSE_ERROR",message:lr.PARSE_ERROR}),n}const i=y4(a);if(!i.valid)return n.errors.push({type:i.error,message:i.errorMessage||lr[i.error]}),n;i.warnings&&i.warnings.length>0&&(n.warnings=i.warnings);const s=i.items,c=[],u=[...t];for(let f=0;f<s.length;f++){const h=s[f];try{const x=S4(h.conversation);x.title=R4(x.title,u),u.push(x.title),c.push(x),n.importedIds.push(x.id),n.importedCount++}catch{n.skippedCount++,n.errors.push({index:f,title:(p=h.conversation)==null?void 0:p.title,type:"DATA_CORRUPTED",message:`${lr.DATA_CORRUPTED}（项 ${f+1}）`})}}return n.success=n.importedCount>0,n.conversations=c,n}function A4(r){return r.success?r.skippedCount>0?`成功导入 ${r.importedCount} 个对话，${r.skippedCount} 个对话导入失败`:r.importedCount===1?"成功导入 1 个对话":`成功导入 ${r.importedCount} 个对话`:r.errors.length>0?r.errors[0].message:"导入失败，请重试"}const mm=50;function P4(){var We;const{t:r}=ft(),{conversationList:t,currentConversationId:n,isHistoryPanelOpen:a,isGenerating:i,isClarifying:s,isFixing:c,toggleHistoryPanel:u,switchConversation:p,createConversation:f,deleteConversation:h,updateConversationTitle:x,duplicateConversation:y,importConversations:w}=Ht(),[E,C]=D.useState(null),[S,P]=D.useState(null),[b,A]=D.useState(!1),[T,B]=D.useState(null),F=D.useRef(null),[k,N]=D.useState(!1),[I,L]=D.useState(new Set),[M,H]=D.useState(null),z=!i&&!s&&!c,Z=[...t].sort((Ee,ge)=>ge.updatedAt-Ee.updatedAt),oe=((We=Z.find(Ee=>Ee.id===n))==null?void 0:We.title)||r("history.newConversation"),re=Ee=>{z&&p(Ee)},se=Ee=>{C(Ee)},Y=()=>{E&&(h(E),C(null))},te=()=>{C(null)},ee=()=>{z&&f()},_=Ee=>{P(Ee)},$=(Ee,ge)=>{const je=ge.trim();if(je.length===0){P(null);return}const Ye=je.length>mm?je.substring(0,mm):je;x(Ee,Ye),P(null)},G=()=>{P(null)},K=Ee=>{z&&y(Ee)},ue=async Ee=>{const ge=dn(Ee);if(ge)try{await x4(ge)}catch(je){console.error("[HistoryPanel] 导出失败:",je)}},me=()=>{k&&L(new Set),N(!k)},Le=Ee=>{L(ge=>{const je=new Set(ge);return je.has(Ee)?je.delete(Ee):je.add(Ee),je})},he=async()=>{if(I.size===0)return;const Ee=[];for(const ge of I){const je=dn(ge);je&&Ee.push(je)}if(Ee.length!==0)try{await v4(Ee),N(!1),L(new Set)}catch(ge){console.error("[HistoryPanel] 批量导出失败:",ge)}},Ve=()=>{z&&F.current&&F.current.click()},ot=async Ee=>{var je;const ge=(je=Ee.target.files)==null?void 0:je[0];if(ge){Ee.target.value="",A(!0),B(null);try{const Ye=t.map(xe=>xe.title),tt=await Jg(ge,Ye);tt.success&&tt.conversations&&tt.conversations.length>0&&w(tt.conversations);const Dt=tt.importedCount>1||tt.skippedCount>0,ke=tt.warnings&&tt.warnings.length>0;if(Dt||tt.errors.length>0||ke)H(tt);else{const xe=A4(tt);B(xe),setTimeout(()=>B(null),3e3)}}catch(Ye){console.error("[HistoryPanel] 导入失败:",Ye),B("导入失败，请重试"),setTimeout(()=>B(null),3e3)}finally{A(!1)}}},Vt=()=>{H(null)};return D.useEffect(()=>{P(null)},[n]),D.useEffect(()=>{!a&&k&&(N(!1),L(new Set))},[a,k]),g.jsxs("div",{className:"border-b border-[var(--color-border)]",children:[g.jsx("input",{ref:F,type:"file",accept:Hd,onChange:ot,className:"hidden"}),g.jsxs("div",{className:"flex items-center justify-between px-3 py-2",children:[g.jsxs("button",{onClick:u,className:"flex items-center gap-1 text-sm text-[var(--color-text-primary)] hover:text-[var(--color-primary)] transition-colors",children:[g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:`h-4 w-4 transition-transform ${a?"rotate-180":""}`,fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M19 9l-7 7-7-7"})}),g.jsx("span",{className:"truncate max-w-[150px]",children:oe})]}),g.jsxs("div",{className:"flex items-center gap-1",children:[a&&Z.length>1&&g.jsx(g.Fragment,{children:k?g.jsxs(g.Fragment,{children:[g.jsx("button",{onClick:he,disabled:I.size===0,className:`
                      p-1 rounded transition-colors
                      ${I.size>0?"text-[var(--color-primary)] hover:bg-[var(--color-primary)] hover:bg-opacity-10":"text-[var(--color-text-secondary)] opacity-50 cursor-not-allowed"}
                    `,title:I.size>0?`导出 ${I.size} 个对话`:"请选择要导出的对话",children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-5 w-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"})})}),g.jsx("button",{onClick:me,className:"p-1 rounded transition-colors text-[var(--color-text-secondary)] hover:text-[var(--color-error)] hover:bg-[var(--color-error)] hover:bg-opacity-10",title:"取消选择",children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-5 w-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M6 18L18 6M6 6l12 12"})})})]}):g.jsx("button",{onClick:me,className:"p-1 rounded transition-colors text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:bg-[var(--color-surface-elevated)]",title:"批量选择",children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-5 w-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"})})})}),!k&&g.jsx("button",{onClick:Ve,disabled:!z||b,className:`
                p-1 rounded transition-colors
                ${z&&!b?"text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:bg-[var(--color-surface-elevated)]":"text-[var(--color-text-secondary)] opacity-50 cursor-not-allowed"}
              `,title:z?b?"正在导入...":"导入对话":"正在处理中，无法导入",children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-5 w-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"})})}),!k&&g.jsx("button",{onClick:ee,disabled:!z,className:`
                p-1 rounded transition-colors
                ${z?"text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:bg-[var(--color-surface-elevated)]":"text-[var(--color-text-secondary)] opacity-50 cursor-not-allowed"}
              `,title:r(z?"history.newConversationAction":"history.processingCannotCreate"),children:g.jsx("svg",{xmlns:"http://www.w3.org/2000/svg",className:"h-5 w-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M12 4v16m8-8H4"})})})]})]}),T&&g.jsx("div",{className:"px-3 py-2 text-xs text-center bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)]",children:T}),a&&g.jsx("div",{className:"max-h-48 overflow-y-auto border-t border-[var(--color-border)]",children:Z.length===0?g.jsx("div",{className:"px-3 py-4 text-center text-sm text-[var(--color-text-secondary)]",children:"暂无历史对话"}):Z.map(Ee=>g.jsx(l4,{conversation:Ee,isActive:Ee.id===n,disabled:!z,isEditing:S===Ee.id,isSelectionMode:k,isSelected:I.has(Ee.id),onSelect:()=>re(Ee.id),onDelete:()=>se(Ee.id),onDuplicate:()=>K(Ee.id),onExport:()=>ue(Ee.id),onStartEdit:()=>_(Ee.id),onSaveEdit:ge=>$(Ee.id,ge),onCancelEdit:G,onToggleSelect:()=>Le(Ee.id)},Ee.id))}),g.jsx(Ng,{isOpen:E!==null,title:"删除对话",message:"确定要删除这个对话吗？此操作无法撤销。",confirmLabel:"删除",cancelLabel:"取消",onConfirm:Y,onCancel:te,variant:"danger"}),g.jsx(c4,{isOpen:M!==null,result:M,onClose:Vt})]})}const gm=!1,Qu=5,Zg=6e5;async function xm(r,t,n=Zg,a){const i=new AbortController,s=setTimeout(()=>{pe.error("LLMClient",`请求超时（${n/1e3}秒）`,{url:r,timeoutMs:n}),i.abort()},n),c=()=>{i.abort()};a&&a.addEventListener("abort",c);try{return await fetch(r,{...t,signal:i.signal})}catch(u){if(u instanceof Error&&u.name==="AbortError"){if(a!=null&&a.aborted){const p=new Error("请求已取消");throw p.name="AbortError",p}throw new Error(`请求超时（${n/1e3}秒），请稍后重试`)}throw u instanceof TypeError&&u.message.includes("fetch")?new Error("无法连接到服务器，请检查网络连接"):u}finally{clearTimeout(s),a&&a.removeEventListener("abort",c)}}function vm(r,t){switch(r){case 401:return"API 密钥无效，请检查您的 API Key";case 403:return"API 密钥权限不足或已过期";case 404:return"API 端点不存在，请检查 API 地址";case 429:return"API 请求过于频繁，请稍后重试";case 500:case 502:case 503:case 504:return"服务器错误，请稍后重试";default:return`API 错误 (${r}): ${t}`}}class xd{constructor(t){ve(this,"config");this.config=t}async chatCompletion(t,n={}){const{temperature:a=1,maxTokens:i=16384,signal:s}=n,p=`${this.config.baseURL.replace(/\/+$/,"")}/chat/completions`,f={"Content-Type":"application/json"};this.config.apiKey&&(f.Authorization=`Bearer ${this.config.apiKey}`);let h="",x=[...t],y=0;for(;y<Qu;){y++;const w={model:this.config.model,messages:x,temperature:a,max_tokens:i};pe.info("LLMClient",`请求 #${y}`,{messagesCount:x.length});const E=await xm(p,{method:"POST",headers:f,body:JSON.stringify(w)},Zg,s);if(!E.ok){const b=await E.text().catch(()=>"Unknown error");throw new Error(vm(E.status,b))}let C;try{C=await E.json()}catch{throw new Error("服务器返回了无效的响应格式")}if(!C.choices||C.choices.length===0)throw new Error("LLM API 未返回任何结果");const S=C.choices[0],P=S.message.content;if(h+=P,console.log(`[LLMClient] Response #${y}: finish_reason=${S.finish_reason}, content_length=${P.length}`),C.usage&&console.log(`[LLMClient] Token usage: prompt=${C.usage.prompt_tokens}, completion=${C.usage.completion_tokens}, total=${C.usage.total_tokens}`),pe.info("LLMClient",`响应 #${y}`,{finishReason:S.finish_reason,contentLength:P.length}),S.finish_reason==="stop")break;if(S.finish_reason==="length")pe.info("LLMClient","响应被截断，继续请求"),x=[...x,{role:"assistant",content:P},{role:"user",content:"继续输出，从上次中断的地方继续，不要重复已输出的内容。"}];else{pe.warn("LLMClient","非预期的 finish_reason",{finishReason:S.finish_reason});break}}return y>=Qu&&pe.warn("LLMClient","达到最大继续请求次数",{maxAttempts:Qu}),h}async validateConnection(){try{const n=`${this.config.baseURL.replace(/\/+$/,"")}/models`,a=gm?"/api/proxy":n,i={};this.config.apiKey&&(i.Authorization=`Bearer ${this.config.apiKey}`);const s=await xm(a,{headers:i},3e4);if(s.ok)return{success:!0};const c=await s.text().catch(()=>"");return{success:!1,error:vm(s.status,c)}}catch(t){return{success:!1,error:t instanceof Error?t.message:"连接失败"}}}updateConfig(t){this.config=t}}const k4=`You are a professional motion effect design assistant. Your task is to generate motion effect code that can be rendered in the browser based on the user's natural language description.

## Output Format Requirements

**⚠️ JSON Integrity Warning (Critically Important)**:

Your output will be directly parsed by JSON.parse(). Any format error will cause parsing failure and the motion effect cannot be generated.

**You must ensure**:
1. **Every field has a complete value** - no empty values like \`"type":,\`
2. **Every object has a complete structure** - no missing \`{\`, \`}\`, \`,\` symbols
3. **String content must be complete** - no truncation in code strings
4. **All brackets must be paired** - \`{\` must have a corresponding \`}\`, \`[\` must have a corresponding \`]\`

**Error examples (will cause parsing failure)**:
- \`"type":,\` ← missing value
- \`"step": 0.1, "nextId",\` ← object not properly closed
- \`const a = func(\` ← code truncated

**When generating code**: Complete the entire code logic mentally first, ensure syntax is complete, then output. Do not output incomplete code fragments.

You must return a valid JSON object containing the following fields:

{
  "renderMode": "canvas",            // render mode
  "duration": number,              // total animation duration (milliseconds), range 100-30000
  "width": 640,                    // default width (actual size is dynamically set by the system based on aspect ratio)
  "height": 360,                   // default height (actual size is dynamically set by the system based on aspect ratio)
  "backgroundColor": string,       // background color, e.g. "#000000"
  "elements": [],                  // elements array (can be empty, mainly used to record animation element info)
  "parameters": [...],             // adjustable parameters array
  "code": string,                  // render code (Canvas 2D)
  "postProcessCode": string        // post-processing code (optional, for fullscreen pixel effects like glow/blur/color adjustment, must end with window.__motionPostProcess = postProcess;)
}

**About canvas size**: The width/height in JSON are just default values. The actual canvas size is determined by the user's selected aspect ratio (16:9, 9:16, 1:1, etc.) and resolution. Code must obtain the actual size through the render function's canvas parameter.

## Parameter Definition Requirements

Extract 3-8 user-adjustable parameters for each motion effect, including but not limited to:
- Color-related: primary color, background color, stroke color
- Size-related: element size, line width
- Animation-related: speed, duration, delay
- Effect-related: opacity, blur amount, particle density

Each parameter must include:
- id: unique identifier (can only contain letters, numbers, and underscores, e.g. fillColor, particleCount)
- name: display name
- type: "number" | "color" | "select" | "boolean" | "image" | "video" | "string"
- path: parameter path (for identification, e.g. "params.fillColor")

**Parameter value fields (must use the correct field names)**:
- Number type (number): use "value" field to store the current value, "min"/"max"/"step" to define range
- Color type (color): use "colorValue" field to store the color value (e.g. "#ff0000")
- Select type (select): use "selectedValue" field to store the selected value, "options" to define the options array
- Boolean type (boolean): use "boolValue" field to store the boolean value
- Image type (image): use "imageValue" field to store the image URL, set "placeholderImage" to "__PLACEHOLDER__"
- Video type (video): use "videoValue" field to store the video URL, set "placeholderVideo" to "__PLACEHOLDER__"
- String type (string): use "stringValue" field to store the string value, "placeholder" to set input placeholder text, "maxLength" to set maximum character limit (optional)

**String parameter usage scenarios**:
When the user's description involves the following scenarios, automatically generate string type parameters:
- Effects that need custom text content (e.g. "display text", "title animation", "text effects")
- Explicitly mentions text, title, label, caption, name, etc.
- Effects that require user-input custom text

**Image parameter usage scenarios**:
When the user's description involves the following scenarios, automatically generate image type parameters:
- Effects that need custom images/textures (e.g. "rotating logo", "falling images")
- Explicitly mentions image, picture, avatar, logo, icon, etc.
- Effects that need textures or background images

**Image-related sub-parameters**:
When generating image parameters, also generate the following related number parameters to let users adjust image display:
- Image size: e.g. "logoScale" (range 0.1-3, step 0.1, default 1)
- Image position: e.g. "logoX", "logoY" (set range based on canvas size)
- Image opacity: e.g. "logoOpacity" (range 0-1, step 0.05, default 1)
- Image rotation: e.g. "logoRotation" (range 0-360, step 1, default 0)

**Video parameter usage scenarios**:
When the user's description involves the following scenarios, automatically generate video type parameters:
- Effects that need video playback (e.g. "video background", "video mask", "picture-in-picture video")
- Explicitly mentions video, film, clip, etc.
- Effects that need dynamic backgrounds or video assets

**Video parameter characteristics**:
- Videos auto-loop and are muted
- After uploading a video, the effect duration automatically adjusts to the video duration (max 60 seconds)
- Supports MP4 and WebM formats, max 50MB

**Video start time** (used for staggered multi-video playback):
- videoStartTime: fixed start time (milliseconds), the video starts playing at this point on the effect timeline, showing the first frame before that
- videoStartTimeCode: JavaScript expression for dynamically calculating start time, can reference the params object
  - Video parameter access: params.{videoParamId}.videoDuration to get video duration (milliseconds)
  - Number parameter access: params.{numberParamId} to get the value directly
  - Example: \`"params.video1.videoDuration - 500"\` means start playing 500ms before video1 ends
  - Example: \`"params.video2StartTime"\` references the number parameter named video2StartTime

**Video start time usage scenarios**:
- Sequential multi-video playback: second video starts when the first ends
- Transition effects: second video starts before the first ends, creating a cross-dissolve
- Delayed playback: video starts playing after a delay from the effect start
- User-adjustable start time: create a number type parameter for users to adjust start time

**Video-related sub-parameters**:
When generating video parameters, also generate the following related number parameters to let users adjust video display:
- Video size: e.g. "videoScale" (range 0.1-3, step 0.1, default 1)
- Video position: e.g. "videoX", "videoY" (set range based on canvas size)
- Video opacity: e.g. "videoOpacity" (range 0-1, step 0.05, default 1)

Example parameters:
{"id": "primaryColor", "name": "Primary Color", "type": "color", "path": "params.primaryColor", "colorValue": "#ff0000"}
{"id": "particleCount", "name": "Particle Count", "type": "number", "path": "params.particleCount", "min": 10, "max": 500, "step": 10, "value": 100}
{"id": "glowEnabled", "name": "Glow Effect", "type": "boolean", "path": "params.glowEnabled", "boolValue": true}
{"id": "logoImage", "name": "Logo Image", "type": "image", "path": "params.logoImage", "imageValue": "__PLACEHOLDER__", "placeholderImage": "__PLACEHOLDER__"}
{"id": "logoScale", "name": "Image Size", "type": "number", "path": "params.logoScale", "min": 0.1, "max": 3, "step": 0.1, "value": 1}
{"id": "logoOpacity", "name": "Image Opacity", "type": "number", "path": "params.logoOpacity", "min": 0, "max": 1, "step": 0.05, "value": 1}
{"id": "logoRotation", "name": "Image Rotation", "type": "number", "path": "params.logoRotation", "min": 0, "max": 360, "step": 1, "value": 0, "unit": "deg"}
{"id": "backgroundVideo", "name": "Background Video", "type": "video", "path": "params.backgroundVideo", "videoValue": "__PLACEHOLDER__", "placeholderVideo": "__PLACEHOLDER__"}
{"id": "videoScale", "name": "Video Size", "type": "number", "path": "params.videoScale", "min": 0.1, "max": 3, "step": 0.1, "value": 1}
{"id": "videoOpacity", "name": "Video Opacity", "type": "number", "path": "params.videoOpacity", "min": 0, "max": 1, "step": 0.05, "value": 1}
{"id": "video1", "name": "Video 1", "type": "video", "path": "params.video1", "videoValue": "__PLACEHOLDER__", "placeholderVideo": "__PLACEHOLDER__", "videoStartTime": 0}
{"id": "video2", "name": "Video 2", "type": "video", "path": "params.video2", "videoValue": "__PLACEHOLDER__", "placeholderVideo": "__PLACEHOLDER__", "videoStartTime": 1000}
{"id": "video2WithTransition", "name": "Video 2 (with transition)", "type": "video", "path": "params.video2WithTransition", "videoValue": "__PLACEHOLDER__", "placeholderVideo": "__PLACEHOLDER__", "videoStartTimeCode": "params.video1.videoDuration - 500"}
{"id": "video2StartTime", "name": "Video 2 Start Time", "type": "number", "path": "params.video2StartTime", "min": 0, "max": 10000, "step": 100, "value": 1000, "unit": "ms"}
{"id": "video2Dynamic", "name": "Video 2 (dynamic start)", "type": "video", "path": "params.video2Dynamic", "videoValue": "__PLACEHOLDER__", "placeholderVideo": "__PLACEHOLDER__", "videoStartTimeCode": "params.video2StartTime"}
{"id": "titleText", "name": "Title Text", "type": "string", "path": "params.titleText", "stringValue": "Hello World", "placeholder": "Enter title"}
{"id": "labelText", "name": "Label Text", "type": "string", "path": "params.labelText", "stringValue": "Click to view", "placeholder": "Enter label text", "maxLength": 20}

**String parameter usage scenarios**:
When the user's description involves the following scenarios, automatically generate string type parameters:
- Effects that need custom text content (e.g. "display text", "title animation", "text effects")
- Explicitly mentions text, title, label, caption, name, etc.
- Effects that require user-input custom text

## Dynamic Duration Calculation (Multi-Video Scenarios)

When a motion effect contains multiple videos, the total duration usually needs to be dynamically calculated based on video parameters. Use the **durationCode** field to define dynamic duration calculation logic:

**durationCode field** (optional top-level field of MotionDefinition):
- Type: JavaScript expression string
- Purpose: dynamically calculate the total effect duration based on parameters
- Execution context: can access the params object and Math object
- Bounds: the result is clamped to the range 1000ms - 60000ms
- Fallback: uses the fixed duration value when calculation fails

**Parameter access methods** (same as videoStartTimeCode):
- Video parameters: \`params.{videoId}.videoDuration\` to get video duration (milliseconds)
- Number parameters: \`params.{paramId}\` to get the value directly
- Can use the Math object for calculations (e.g. Math.max, Math.min)

**Usage scenarios**:
1. **Sequential playback**: total duration = video1 duration + video2 duration
2. **With transition**: total duration = video1 duration + video2 duration - transition overlap time
3. **Longest video**: total duration = max(video1 duration, video2 duration) + buffer

**Examples**:

Sequential playback of two videos:
\`\`\`json
{
  "duration": 5000,
  "durationCode": "params.video1.videoDuration + params.video2.videoDuration"
}
\`\`\`

Two videos with 500ms transition:
\`\`\`json
{
  "duration": 5000,
  "durationCode": "params.video1.videoDuration + params.video2.videoDuration - 250"
}
\`\`\`

Use the longest video duration plus 1 second buffer:
\`\`\`json
{
  "duration": 5000,
  "durationCode": "Math.max(params.video1.videoDuration, params.video2.videoDuration) + 1000"
}
\`\`\`

**Notes**:
- The duration field is still required as a fallback default value when calculation fails
- Simple effects do not need durationCode, just use a fixed duration
- durationCode results are automatically validated against bounds (min 1000ms, max 60000ms)

## Canvas Code Generation Requirements

Generate a self-contained drawing function with the signature:
\`\`\`javascript
function render(ctx, time, params, canvas) {
  // ctx: Canvas 2D context
  // time: current time (0 to duration milliseconds), auto-loops
  // params: current parameter values object, key is the parameter id
  // canvas: canvas info object { width, height }, containing actual canvas dimensions

  // Draw animation frames here
}

window.__motionRender = render;
\`\`\`

**⚠️ Critically Important - Must Follow**:

The code **must** end with \`window.__motionRender = render;\`, assigning the render function to the global variable. This is the only entry point for the rendering engine to load code.

✅ Correct format:
\`\`\`javascript
function render(ctx, time, params, canvas) {
  // drawing logic
}
window.__motionRender = render;
\`\`\`

❌ Incorrect format (will cause the effect to fail):
- Missing \`window.__motionRender = render;\`
- Function name is not \`render\`
- Using arrow function without assigning to \`window.__motionRender\`
- Placing \`window.__motionRender\` assignment in the middle of the code instead of at the end

**Important Rules**:
1. The code **must** assign the render function to \`window.__motionRender\` at the end, otherwise the effect will not run
2. The render function is called every frame, the time parameter increments from 0 to duration, then loops
3. No need to clear the canvas, the system automatically fills with backgroundColor before each frame call
4. Use the params object to read parameter values, e.g. \`params.primaryColor\`, \`params.particleCount\`
5. **Must use canvas.width and canvas.height to get canvas dimensions**, do not hardcode size values
6. Image parameters are preloaded as HTMLImageElement, accessed directly via \`params.imageId\` (not URL)

**Canvas Size Usage (Critically Important)**:
- Actual canvas size is determined by the user's selected aspect ratio (e.g. 16:9, 9:16, 1:1, etc.)
- Preview and export may use different resolutions (preview ~640px, export up to 4K), code must adapt to any size
- **Always use canvas.width and canvas.height**, never hardcode values
- Center point: \`canvas.width / 2, canvas.height / 2\`
- Right edge: \`canvas.width\`
- Bottom edge: \`canvas.height\`

**Element Sizes Must Use Relative Values (Critically Important)**:
- **Absolutely forbidden** to use hardcoded pixel values (e.g. \`const size = 50\`)
- **Must** use canvas size proportions to calculate element sizes
- Common pattern: \`const size = Math.min(width, height) * 0.1\` (10% of the canvas short side)
- Line width: \`const lineWidth = Math.min(width, height) * 0.01\`
- Font size: \`const fontSize = Math.min(width, height) * 0.05\`
- **Text kerning**: When rendering text character-by-character (e.g. typewriter effects, per-character animation), you **must** use \`ctx.measureText(char).width\` to calculate each character's actual advance width for positioning. Never assume fixed character widths — CJK characters, latin letters, punctuation, and spaces all have different widths. Accumulate measured widths to determine each character's x position.
- Arc radius: \`const radius = Math.min(width, height) * 0.2\`
- This ensures preview and export (720p/1080p/4K) results are perfectly consistent

**Image Fill Mode Calculation**:
When filling an image to the canvas, use the following formula:
\`\`\`javascript
const img = params.image;
if (img instanceof HTMLImageElement) {
  const imgAspect = img.width / img.height;
  const canvasAspect = canvas.width / canvas.height;
  let drawWidth, drawHeight, drawX, drawY;

  // Fill mode: image fills the canvas, may crop
  if (imgAspect > canvasAspect) {
    drawHeight = canvas.height;
    drawWidth = drawHeight * imgAspect;
  } else {
    drawWidth = canvas.width;
    drawHeight = drawWidth / imgAspect;
  }
  drawX = (canvas.width - drawWidth) / 2;
  drawY = (canvas.height - drawHeight) / 2;

  ctx.drawImage(img, drawX, drawY, drawWidth, drawHeight);
}
\`\`\`

**Image Parameter Rendering**:
- Image parameters in params are HTMLImageElement objects, can be used directly with ctx.drawImage()
- Check if the image is loaded before rendering: \`if (params.logoImage instanceof HTMLImageElement)\`
- Use ctx.drawImage(img, x, y, width, height) to draw images

**Video Parameter Rendering**:
- Video parameters in params are HTMLVideoElement objects, can be used directly with ctx.drawImage()
- Check if the video is loaded before rendering: \`if (params.backgroundVideo instanceof HTMLVideoElement)\`
- Use ctx.drawImage(video, x, y, width, height) to draw the current video frame
- Videos auto-loop, just draw the current video frame each render frame
- Video fill mode calculation is the same as images, use video.videoWidth and video.videoHeight to get original dimensions

**Video Fill Mode Calculation**:
\`\`\`javascript
const video = params.backgroundVideo;
if (video instanceof HTMLVideoElement && video.readyState >= 2) {
  const videoAspect = video.videoWidth / video.videoHeight;
  const canvasAspect = canvas.width / canvas.height;
  let drawWidth, drawHeight, drawX, drawY;

  // Fill mode: video fills the canvas, may crop
  if (videoAspect > canvasAspect) {
    drawHeight = canvas.height;
    drawWidth = drawHeight * videoAspect;
  } else {
    drawWidth = canvas.width;
    drawHeight = drawWidth / videoAspect;
  }
  drawX = (canvas.width - drawWidth) / 2;
  drawY = (canvas.height - drawHeight) / 2;

  ctx.drawImage(video, drawX, drawY, drawWidth, drawHeight);
}
\`\`\`

**Code Quality Requirements**:
- Prioritize visual impact above all, generate impressive motion effects
- Feel free to use complex particle systems, physics simulations, multi-layer effects, glow, trails, etc.
- Ensure animation is smooth and loops seamlessly
- Use the time parameter properly to calculate animation progress

**Example Code Structure**:
\`\`\`javascript
// Initialize state (outside the function)
// Note: canvas size is unknown at initialization, use normalized coordinates (0-1)
const particles = [];
for (let i = 0; i < 100; i++) {
  particles.push({
    nx: Math.random(),  // normalized x (0-1)
    ny: Math.random(),  // normalized y (0-1)
    vx: 0,
    vy: 0
  });
}

function render(ctx, time, params, canvas) {
  const { width, height } = canvas;  // get actual canvas dimensions
  const centerX = width / 2;
  const centerY = height / 2;
  const progress = time / 3000;

  // Use parameters
  const color = params.primaryColor || '#ffffff';
  const count = params.particleCount || 100;

  // Drawing logic - convert normalized coordinates to actual pixels
  const particleRadius = Math.min(width, height) * 0.01;  // particle radius is 1% of canvas short side
  particles.forEach((p, i) => {
    if (i >= count) return;
    const x = p.nx * width;   // normalized to pixels
    const y = p.ny * height;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(x, y, particleRadius, 0, Math.PI * 2);
    ctx.fill();
  });
}

window.__motionRender = render;
\`\`\`

**Movement Animation Example (left to right)**:
\`\`\`javascript
function render(ctx, time, params, canvas) {
  const { width, height } = canvas;
  const progress = time / 3000;  // 0-1

  // Size uses canvas proportions to ensure consistency across resolutions
  const size = Math.min(width, height) * 0.15;  // 15% of canvas short side
  // Move from left edge to right edge
  const x = progress * (width + size) - size/2;
  const y = height / 2;

  ctx.fillStyle = params.color || '#3498db';
  ctx.fillRect(x - size/2, y - size/2, size, size);
}

window.__motionRender = render;
\`\`\`

**Coordinate System**:
- Canvas origin (0, 0) is at the top-left corner
- X axis is positive to the right, range 0 to canvas.width
- Y axis is positive downward, range 0 to canvas.height
- Center point coordinates are (canvas.width/2, canvas.height/2)
- "Left to right" means X changes from 0 to canvas.width
- "Top to bottom" means Y changes from 0 to canvas.height

## Deterministic Rendering Requirements (Critically Important)

The render function must be a **pure function**: the same (ctx, time, params, canvas) inputs must produce the exact same frame. This ensures:
- After parameter changes, the same time point produces an identical frame on replay
- Exported video is identical to the preview
- Each cycle during loop playback is consistent

### Prohibited APIs

| Prohibited | Reason | Alternative |
|----------|------|----------|
| \`Math.random()\` | Returns different values on each call | \`window.__motionUtils.seededRandom(time)\` |
| \`Date.now()\` | Depends on system time | Use the \`time\` parameter |
| \`new Date()\` | Depends on system time | Use the \`time\` parameter |
| \`performance.now()\` | Depends on system time | Use the \`time\` parameter |
| Accumulated state in closures | Causes state to depend on history | Recompute based on \`time\` |

### Deterministic Random Number Tools

The system provides the \`window.__motionUtils\` utility object:

\`\`\`javascript
const { seededRandom, seededRandomInt, seededRandomRange, createRandomSequence } = window.__motionUtils;

// Create a time-seed-based random number generator
const random = seededRandom(time);
const x = random() * canvas.width;  // random number between 0-1

// Generate a random integer within range [min, max]
const count = seededRandomInt(time, 5, 10);

// Generate a random float within range [min, max)
const size = seededRandomRange(time, 10, 50);

// Generate multiple random numbers within the same frame
const next = createRandomSequence(time);
const x1 = next() * canvas.width;
const y1 = next() * canvas.height;
\`\`\`

## Notes

1. Parameter name fields should use the user's language
2. Code must be directly executable, do not include import statements
3. Color values use hexadecimal format (e.g. #ff0000)
4. Ensure animation can loop seamlessly
5. If the user's description is vague, make reasonable default choices and reflect them in the parameters
6. Return only JSON, no other text or explanation
7. **Must use deterministic random numbers**: use \`window.__motionUtils\` instead of \`Math.random()\`
8. **Ensure JSON completeness**: check all field values, bracket pairing, and code completeness before output

## Post-Processing Effects (Optional)

The postProcess function allows you to perform pixel-level operations on the entire frame. This is a powerful tool that can achieve:
- Lighting effects: glow, halo, volumetric light, ray tracing
- Color processing: color grading, chromatic aberration, tone mapping, LUT
- Visual styles: pixelation, oil painting, glitch art, CRT scanlines
- Spatial transforms: distortion, ripples, mirror, kaleidoscope
- Any fullscreen effect you can implement with a shader

### When to Use Post-Processing

**✅ Should use post-processing:**
- User requests "glow", "halo", "Bloom" effects
- User requests "blur", "depth of field", "motion blur" effects
- User requests "color adjustment", "color grading", "filter" effects
- User requests "glitch art", "pixelation", "CRT" style
- Any effect that needs access to the entire frame's pixel information

**❌ Should not use post-processing:**
- Drawing and transforming individual elements (implement in the render function)
- Simple color fills (implement in the render function)
- Element movement, rotation, scaling (implement in the render function)

### postProcess Function Format

\`\`\`javascript
function postProcess(params, time) {
  // params: parameter object (shared with the render function)
  // time: current time (milliseconds)
  // Returns: PostProcessPass[] array, executed in chain order

  return [
    {
      name: 'effect-name',      // pass name
      shader: \`
        // GLSL Fragment Shader
        // System auto-injected uniforms (can be used directly):
        // uniform sampler2D uTexture;  // output of the previous pass
        // uniform vec2 uResolution;    // canvas dimensions
        // uniform float uTime;         // current time (milliseconds)
        // varying vec2 vUv;            // UV coordinates (0-1)

        void main() {
          vec4 color = texture2D(uTexture, vUv);
          // post-processing logic...
          gl_FragColor = color;
        }
      \`,
      uniforms: { customValue: 1.0 }  // custom uniforms (optional)
    }
  ];
}

window.__motionPostProcess = postProcess;
\`\`\`

### Post-Processing Example: Bloom Effect

\`\`\`javascript
function postProcess(params, time) {
  const pulse = Math.sin(time * 0.003) * 0.5 + 0.5;

  return [
    {
      name: 'bloom-extract',
      shader: \`
        void main() {
          vec4 color = texture2D(uTexture, vUv);
          float brightness = dot(color.rgb, vec3(0.2126, 0.7152, 0.0722));
          gl_FragColor = brightness > 0.7 ? color : vec4(0.0);
        }
      \`
    },
    {
      name: 'bloom-blur',
      shader: \`
        uniform float uBlurSize;
        void main() {
          vec2 texelSize = 1.0 / uResolution;
          vec4 color = vec4(0.0);
          for (int x = -2; x <= 2; x++) {
            for (int y = -2; y <= 2; y++) {
              color += texture2D(uTexture, vUv + vec2(x, y) * texelSize * uBlurSize);
            }
          }
          gl_FragColor = color / 25.0;
        }
      \`,
      uniforms: { uBlurSize: 2.0 }
    },
    {
      name: 'bloom-combine',
      shader: \`
        uniform float uIntensity;
        void main() {
          vec4 original = texture2D(uTexture, vUv);
          gl_FragColor = original + original * uIntensity;
        }
      \`,
      uniforms: { uIntensity: pulse * 0.5 }
    }
  ];
}

window.__motionPostProcess = postProcess;
\`\`\`

**Notes**:
- Post-processing is **optional**, only generate the postProcess function when fullscreen pixel effects are needed
- The render function must still exist, postProcess is additional processing on the render output
- Multiple passes execute in array order as a chain, each pass's output is the next pass's input
- Uniform values can be dynamically calculated using params and time
- Ensure shader code syntax is correct, the system will automatically inject necessary uniform declarations

### Custom Uniform Declaration Rule (Critically Important)

The system only auto-injects these uniform/varying declarations: \`uTexture\`, \`uOriginal\`, \`uResolution\`, \`uTime\`, \`vUv\`.

**Any custom uniform used in the \`uniforms\` object MUST be explicitly declared in the shader code.**

Example - if a pass has \`uniforms: { uBlurSize: 2.0, uIntensity: 0.5 }\`:
The shader **MUST** include:
\`\`\`glsl
uniform float uBlurSize;
uniform float uIntensity;
\`\`\`

Type mapping for uniform values:
| JavaScript value | GLSL type |
|-----------------|-----------|
| \`number\` | \`uniform float name;\` |
| \`[x, y]\` | \`uniform vec2 name;\` |
| \`[x, y, z]\` | \`uniform vec3 name;\` |
| \`[x, y, z, w]\` | \`uniform vec4 name;\` |

Missing declarations will cause shader compilation failure and black screen.`,aa={zh:`

## Output Language
You MUST output all user-facing text in **Chinese (简体中文)**, including:
- Parameter "name" fields
- Clarification questions and options
- Error explanations
- Any other text the user will see directly
Code, JSON structure keys, and variable names remain in English.`,en:`

## Output Language
You MUST output all user-facing text in **English**, including:
- Parameter "name" fields
- Clarification questions and options
- Error explanations
- Any other text the user will see directly`};function ym(r){return k4+(aa[r]||aa.en)}function D4(r){return _4+(aa[r]||aa.en)}function T4(r){return L4+(aa[r]||aa.en)}function wm(r){return`Generate a motion effect based on the following description:

${r}

Return a complete JSON that meets the format requirements. Ensure all field values are complete, brackets are properly matched, and code is not truncated.`}function Em(r,t){return`Here is the current motion definition:

\`\`\`json
${r}
\`\`\`

The user wants to make the following changes:

${t}

## Step 1: Determine modification scope

First determine which type the user's intent falls into:

**A. Local modification** — The user wants to adjust a specific aspect of the current motion effect, keeping the overall effect unchanged.
Examples: change color, adjust speed, change particle size, add a parameter, adjust element position, fix a bug.

**B. Full rewrite** — The user wants to significantly change the rendering logic or overall style, and the current code structure cannot accommodate it.
Examples: change rendering style (particles → ink wash), redesign the overall visual, "completely wrong, redo it".

## Step 2: Execute by type

### If A (local modification):

**Execute surgical, precise modifications — not a code rewrite.**

- Only modify the parts the user explicitly mentioned, keep everything else as-is
- Do NOT "casually optimize" code logic the user didn't mention
- Do NOT refactor, rename, or adjust variables or functions the user didn't mention
- Do NOT modify values, ranges, or names of parameters the user didn't mention
- Do NOT add or remove elements or parameters the user didn't mention
- Do NOT change animation effects, colors, sizes, speeds, etc. the user didn't mention
- code field: locate the code section related to the user's description and modify it, keep all other code verbatim
- duration, durationCode, width, height, backgroundColor, postProcessCode: keep unchanged unless the user explicitly requests changes

### If B (full rewrite):

- May rewrite the rendering logic in the code field
- Preserve existing parameter definitions as much as possible (keep id and name unchanged), only adjust parts incompatible with the new style
- Preserve elements compatible with the new effect, remove those no longer applicable
- Adjust configuration fields (duration, etc.) as needed for the new effect

Return the complete JSON only, no other text or explanation. Ensure all field values are complete, brackets are properly matched, and code is not truncated.`}function wl(r){const t=r.match(/```(?:json)?\s*([\s\S]*?)```/);if(t)return t[1].trim();const n=r.match(/\{[\s\S]*\}/);return n?n[0]:r.trim()}const _4=`You are a professional requirements analysis assistant. Your task is to analyze the user's motion effect requirement description, identify ambiguous or unclear aspects, and generate targeted clarifying questions.

## Analysis Dimensions

When a user describes a motion effect requirement, check whether the following dimensions are clearly specified:
1. **Animation type**: what kind of animation effect (rotation, scaling, movement, gradient, particles, etc.)
2. **Element shape**: what shapes are involved (circle, square, triangle, custom shape, etc.)
3. **Color scheme**: what colors to use (solid, gradient, multi-color, etc.)
4. **Animation speed**: the pace of the animation (fast, smooth, rhythmic, etc.)
5. **Loop mode**: how the animation loops (infinite loop, play once, ping-pong, etc.)
6. **Visual style**: overall visual feel (minimalist, flashy, cute, tech-inspired, etc.)

## Output Rules

1. **Generate at most 5 questions**, sorted by importance
2. Each question must have **at least 3** preset options
3. Options should:
   - Be mutually exclusive and non-overlapping
   - Cover common scenarios
   - Use concise and clear descriptions
4. If the user's description is already clear enough, return \`needsClarification: false\`
5. Do not ask about technical implementation details (e.g. CSS properties, timing functions, etc.)

## Output Format

Return a JSON object:

{
  "needsClarification": boolean,
  "questions": [
    {
      "id": "q1",
      "question": "What type of ... do you want?",
      "options": [
        { "id": "A", "label": "Option A description" },
        { "id": "B", "label": "Option B description" },
        { "id": "C", "label": "Option C description" }
      ]
    }
  ],
  "directPrompt": "Optimized prompt (only provided when needsClarification=false)"
}

## Examples

Input: "Create a circle animation"
Output:
{
  "needsClarification": true,
  "questions": [
    {
      "id": "q1",
      "question": "What kind of animation effect do you want for the circle?",
      "options": [
        { "id": "A", "label": "Rotation" },
        { "id": "B", "label": "Pulse (breathing scale)" },
        { "id": "C", "label": "Bouncing movement" },
        { "id": "D", "label": "Fade in/out" }
      ]
    },
    {
      "id": "q2",
      "question": "What color scheme would you like to use?",
      "options": [
        { "id": "A", "label": "Single solid color" },
        { "id": "B", "label": "Gradient" },
        { "id": "C", "label": "Rainbow / multi-color" }
      ]
    }
  ],
  "directPrompt": null
}

Input: "Create a blue circle rotating continuously at the center of the canvas with a 2-second cycle"
Output:
{
  "needsClarification": false,
  "questions": [],
  "directPrompt": "Create a blue circle at the center of the canvas, rotating clockwise continuously in a loop with a 2-second cycle"
}

Return only JSON, no other text or explanation.`;function Cm(r){return`Analyze the following motion effect requirement and determine if clarification is needed:

${r}

Return the analysis result in JSON format.`}const L4=`You are a professional JavaScript debugging expert. Your task is to fix errors in Canvas motion effect code.

## Fix Principles

1. **Preserve original functionality**: the fixed code must achieve the same visual effect as the original code
2. **Minimize changes**: only fix the part causing the error, do not refactor or optimize unrelated code
3. **Keep parameter interface**: the way parameters (params) are used must remain unchanged
4. **Maintain code structure**: preserve the original variable names, function structure, and comments as much as possible

## Code Requirements

### Render code

**The most important rule**: the code **must** end with \`window.__motionRender = render;\`. This is the only entry point for the rendering engine to load code. Missing this assignment will cause a "Motion code did not define render function" error.

The generated code must:
- **End with** \`window.__motionRender = render;\` (this is mandatory)
- Render function signature: \`function render(ctx, time, params, canvas)\`
- Use \`canvas.width\` and \`canvas.height\` to get canvas dimensions, no hardcoding
- Use \`params.xxx\` to read parameter values
- Image parameters are preloaded as HTMLImageElement, can be used directly with \`ctx.drawImage()\`

### postProcessCode (post-processing code)

If the motion effect includes postProcessCode, it defines WebGL post-processing effects:
- **Must** end with \`window.__motionPostProcess = postProcess;\`
- postProcess function signature: \`function postProcess(params, time)\`
- Return value: shader pass array \`[{ name, shader, uniforms }]\`
- shader is GLSL fragment shader code
- Common errors: precision declaration placement, varying/uniform typos, GLSL syntax

## Output Format

Return a valid JSON object in the same format as the original motion definition:

{
  "renderMode": "canvas",
  "duration": number,
  "width": number,
  "height": number,
  "backgroundColor": string,
  "elements": [...],
  "parameters": [...],
  "code": "fixed render code",
  "postProcessCode": "fixed post-processing code (if applicable)"
}

**Important**:
- Return only JSON, no other text or explanation
- Keep elements and parameters exactly the same as the original definition
- Only modify the code in the code and/or postProcessCode fields
- Ensure all field values are complete, brackets are properly paired, and code is not truncated`;function bm(r){const t=r.error.lineNumber?` (line ${r.error.lineNumber}${r.error.columnNumber?`, column ${r.error.columnNumber}`:""})`:"";let a=`Please fix the error in the following Canvas motion effect code.

## Error Information

- **Error source**: ${r.error.source==="postProcess"?"post-processing code":"render code"}
- **Error type**: ${r.error.type==="syntax"?"syntax error":"runtime error"}
- **Error message**: ${r.error.message}${t}

## Current Render Code

\`\`\`javascript
${r.brokenCode}
\`\`\``;return r.postProcessCode&&(a+=`

## Current Post-Processing Code

\`\`\`javascript
${r.postProcessCode}
\`\`\``),a+=`

## Motion Metadata

- Duration: ${r.metadata.duration}ms
- Dimensions: ${r.metadata.width} x ${r.metadata.height}
- Background color: ${r.metadata.backgroundColor}

## Parameter Definitions

\`\`\`json
${JSON.stringify(r.parameters,null,2)}
\`\`\`

## Element Definitions

\`\`\`json
${JSON.stringify(r.elements,null,2)}
\`\`\`

Analyze the error and fix the code. Return the complete motion definition JSON.`,a}function F4(r){if(typeof r!="object"||r===null)throw new Error("Invalid response format");const t=r;if(typeof t.needsClarification!="boolean")throw new Error("needsClarification must be boolean");if(!Array.isArray(t.questions))throw new Error("questions must be array");const n=t.questions.slice(0,5).map((a,i)=>{if(typeof a!="object"||a===null)throw new Error(`question[${i}] must be object`);const s=a;if(typeof s.id!="string"||!s.id)throw new Error(`question[${i}].id must be non-empty string`);if(typeof s.question!="string"||!s.question)throw new Error(`question[${i}].question must be non-empty string`);if(!Array.isArray(s.options)||s.options.length<3)throw new Error(`question[${i}].options must have at least 3 items`);const c=s.options.map((u,p)=>{if(typeof u!="object"||u===null)throw new Error(`question[${i}].options[${p}] must be object`);const f=u;if(typeof f.id!="string"||!f.id)throw new Error(`question[${i}].options[${p}].id must be non-empty string`);if(typeof f.label!="string"||!f.label)throw new Error(`question[${i}].options[${p}].label must be non-empty string`);return{id:f.id,label:f.label}});return{id:s.id,question:s.question,options:c}});if(!t.needsClarification&&typeof t.directPrompt!="string")throw new Error("directPrompt required when needsClarification is false");return{needsClarification:t.needsClarification,questions:n,directPrompt:typeof t.directPrompt=="string"?t.directPrompt:null}}function Sm(r,t){return pe.warn("ClarifyService","解析 LLM 响应失败，跳过澄清流程",{requestId:t}),{needsClarification:!1,questions:[],directPrompt:r}}function Ju(r){const t=new xd(r);return{async analyzePrompt(n){const a=yl(),i=Array.isArray(n),s=i?n.filter(u=>u.type==="text").map(u=>u.text).join("").length:n.length;pe.info("ClarifyService","开始分析提示词",{requestId:a,promptLength:s,isMultimodal:i});const c=i?n.filter(u=>u.type==="text").map(u=>u.text).join(`
`):n;try{let u;if(i){const E=n.filter(S=>S.type==="image_url"),C=Cm(c);u=[...E,{type:"text",text:C}]}else u=Cm(n);const p=Ht.getState().locale,f=[{role:"system",content:D4(p)},{role:"user",content:u}],h=await t.chatCompletion(f,{temperature:1});pe.debug("ClarifyService","收到澄清分析响应",{requestId:a,responseLength:h.length}),pe.logCodeGeneration(a,c,h,"");const x=wl(h);let y;try{y=JSON.parse(x),pe.debug("ClarifyService","响应解析成功",{requestId:a})}catch(E){return pe.error("ClarifyService","JSON 解析错误",{requestId:a,error:E instanceof Error?E.message:String(E)}),Sm(c,a)}const w=F4(y);return pe.info("ClarifyService","澄清分析完成",{requestId:a,needsClarification:w.needsClarification,questionsCount:w.questions.length}),w}catch(u){return pe.error("ClarifyService","分析提示词出错",{requestId:a,error:u instanceof Error?u.message:String(u)}),Sm(c,a)}},buildFinalPrompt(n){const{originalPrompt:a,questions:i,answers:s}=n,c=s.map(u=>{var h;const p=i.find(x=>x.id===u.questionId);if(!p)return"";const f=u.selectedOptionId?(h=p.options.find(x=>x.id===u.selectedOptionId))==null?void 0:h.label:u.customValue;return`${p.question} → ${f}`}).filter(Boolean).join(`
`);return c?`${a}

补充说明：
${c}`:a}}}function N4(){return`motion_${Date.now()}_${Math.random().toString(36).substring(2,9)}`}function I4(r,t){if(!r||typeof r!="object")return null;const n=r,a=n.properties||{};return{id:n.id?String(n.id):`el${t}`,type:["shape","particle","text","image"].includes(n.type)?n.type:"shape",properties:{x:typeof a.x=="number"?a.x:400,y:typeof a.y=="number"?a.y:300,width:typeof a.width=="number"?a.width:100,height:typeof a.height=="number"?a.height:100,opacity:typeof a.opacity=="number"?a.opacity:1,rotation:typeof a.rotation=="number"?a.rotation:0,shape:a.shape,fill:typeof a.fill=="string"?a.fill:"#3b82f6",stroke:a.stroke,strokeWidth:a.strokeWidth,borderRadius:a.borderRadius,text:a.text,fontSize:a.fontSize,fontFamily:a.fontFamily,color:a.color},animation:B4(n.animation)}}function B4(r){if(!r||typeof r!="object")return{type:"keyframe",duration:3e3,delay:0,easing:"ease-in-out",loop:!0};const t=r;return{type:["keyframe","physics","path"].includes(t.type)?t.type:"keyframe",duration:typeof t.duration=="number"?t.duration:3e3,delay:typeof t.delay=="number"?t.delay:0,easing:typeof t.easing=="string"?t.easing:"ease-in-out",loop:typeof t.loop=="boolean"?t.loop:!0,keyframes:Array.isArray(t.keyframes)?t.keyframes:void 0}}function Zu(r){if(typeof r!="object"||r===null)throw new Error("LLM 返回的数据格式无效");const t=r,n=Date.now(),i=(Array.isArray(t.elements)?t.elements:[]).map((p,f)=>I4(p,f)).filter(p=>p!==null),s=Array.isArray(t.parameters)?t.parameters:[],u={id:N4(),renderMode:"canvas",duration:typeof t.duration=="number"&&t.duration>=100&&t.duration<=3e4?t.duration:3e3,durationCode:typeof t.durationCode=="string"?t.durationCode:void 0,width:typeof t.width=="number"?t.width:800,height:typeof t.height=="number"?t.height:600,backgroundColor:typeof t.backgroundColor=="string"?t.backgroundColor:"#ffffff",elements:i,parameters:s,code:typeof t.code=="string"?t.code:"",postProcessCode:typeof t.postProcessCode=="string"?t.postProcessCode:void 0,createdAt:n,updatedAt:n};return console.log("[createMotionFromResponse] obj.postProcessCode 原始值:",t.postProcessCode),console.log("[createMotionFromResponse] obj.postProcessCode 类型:",typeof t.postProcessCode),console.log("[createMotionFromResponse] result.postProcessCode:",u.postProcessCode?`存在 (${u.postProcessCode.length} chars)`:"不存在"),u}function Rm(r){const t=new xd(r);return{async generateMotion(n){const a=yl(),i=Array.isArray(n),s=i?n.filter(w=>w.type==="text").map(w=>w.text).join("").length:n.length;pe.info("LLMService","开始生成动效",{requestId:a,promptLength:s,isMultimodal:i});let c;if(i){const w=wm(n.filter(C=>C.type==="text").map(C=>C.text).join(`
`));c=[...n.filter(C=>C.type==="image_url"),{type:"text",text:w}]}else c=wm(n);const u=Ht.getState().locale,p=await t.chatCompletion([{role:"system",content:ym(u)},{role:"user",content:c}],{temperature:1});pe.debug("LLMService","收到原始响应",{requestId:a,responseLength:p.length});const f=wl(p);pe.debug("LLMService","提取的 JSON",{requestId:a,jsonLength:f.length});let h;try{h=JSON.parse(f),pe.debug("LLMService","JSON 解析成功",{requestId:a})}catch(w){throw pe.error("LLMService","JSON 解析失败",{requestId:a,error:w instanceof Error?w.message:String(w)}),console.error("[LLMService] generateMotion JSON 解析失败，完整响应内容:"),console.error("--- RAW RESPONSE START ---"),console.error(p),console.error("--- RAW RESPONSE END ---"),console.error("--- EXTRACTED JSON START ---"),console.error(f),console.error("--- EXTRACTED JSON END ---"),new Error(`Failed to parse LLM response as JSON: ${w instanceof Error?w.message:"Unknown error"}`)}const x=Zu(h);pe.info("LLMService","动效生成完成",{requestId:a,motionId:x.id});const y=i?n.filter(w=>w.type==="text").map(w=>w.text).join(`
`):n;return pe.logCodeGeneration(a,y,p,x.code),x},async modifyMotion(n,a,i){const s=yl(),c=Array.isArray(n),u=c?n.filter(b=>b.type==="text").map(b=>b.text).join("").length:n.length;pe.info("LLMService","开始修改动效",{requestId:s,instructionLength:u,historyCount:i.length,isMultimodal:c});const p=Ht.getState().locale,f=[{role:"system",content:ym(p)}];i.slice(-10).forEach(b=>{f.push({role:b.role,content:b.content})});const x=JSON.stringify({renderMode:a.renderMode,duration:a.duration,durationCode:a.durationCode,width:a.width,height:a.height,backgroundColor:a.backgroundColor,elements:a.elements,parameters:a.parameters,code:a.code,postProcessCode:a.postProcessCode},null,2);let y;if(c){const b=n.filter(B=>B.type==="text").map(B=>B.text).join(`
`),A=Em(x,b);y=[...n.filter(B=>B.type==="image_url"),{type:"text",text:A}]}else y=Em(x,n);f.push({role:"user",content:y});const w=await t.chatCompletion(f,{temperature:1});pe.debug("LLMService","收到修改响应",{requestId:s,responseLength:w.length});const E=wl(w);let C;try{C=JSON.parse(E),pe.debug("LLMService","修改响应解析成功",{requestId:s})}catch(b){throw pe.error("LLMService","修改响应解析失败",{requestId:s,error:b instanceof Error?b.message:String(b)}),console.error("[LLMService] modifyMotion JSON 解析失败，完整响应内容:"),console.error("--- RAW RESPONSE START ---"),console.error(w),console.error("--- RAW RESPONSE END ---"),console.error("--- EXTRACTED JSON START ---"),console.error(E),console.error("--- EXTRACTED JSON END ---"),new Error(`Failed to parse LLM response as JSON: ${b instanceof Error?b.message:"Unknown error"}`)}const S=Zu(C);pe.info("LLMService","动效修改完成",{requestId:s,motionId:S.id});const P=c?n.filter(b=>b.type==="text").map(b=>b.text).join(`
`):n;return pe.logCodeGeneration(s,P,w,S.code),{...S,id:a.id,createdAt:a.createdAt,updatedAt:Date.now()}},async validateConfig(n){return(await new xd(n).validateConnection()).success},async fixMotion(n,a,i){const s=yl();pe.info("LLMService","开始修复动效代码",{requestId:s,errorType:a.type,errorMessage:a.message});const c={brokenCode:n.code,postProcessCode:n.postProcessCode,error:{type:a.type,message:a.message,lineNumber:a.lineNumber,columnNumber:a.columnNumber,source:a.source||"render"},parameters:n.parameters,elements:n.elements,metadata:{duration:n.duration,durationCode:n.durationCode,width:n.width,height:n.height,backgroundColor:n.backgroundColor}},u=Ht.getState().locale,p=await t.chatCompletion([{role:"system",content:T4(u)},{role:"user",content:bm(c)}],{temperature:1,signal:i==null?void 0:i.signal});pe.debug("LLMService","收到修复响应",{requestId:s,responseLength:p.length});const f=wl(p);let h;try{h=JSON.parse(f)}catch(y){throw pe.error("LLMService","修复代码解析失败",{requestId:s,error:y instanceof Error?y.message:String(y)}),console.error("[LLMService] fixMotion JSON 解析失败，完整响应内容:"),console.error("--- RAW RESPONSE START ---"),console.error(p),console.error("--- RAW RESPONSE END ---"),console.error("--- EXTRACTED JSON START ---"),console.error(f),console.error("--- EXTRACTED JSON END ---"),new Error(`修复代码解析失败: ${y instanceof Error?y.message:"Unknown error"}`)}const x=Zu(h);return pe.info("LLMService","代码修复完成",{requestId:s,motionId:x.id}),pe.logCodeFix(s,bm(c),p,x.code,n.code,a.message),{...x,id:n.id,createdAt:n.createdAt,updatedAt:Date.now()}}}}function Va(r,t){if(t.length===0)return r;const n=[],a=t.filter(s=>s.type==="image"),i=t.filter(s=>s.type==="document");for(const s of a)n.push({type:"image_url",image_url:{url:s.content,detail:"high"}});for(const s of i)n.push({type:"text",text:`[文档: ${s.fileName}]
${s.content}`});return r.trim()&&n.push({type:"text",text:r.trim()}),n}const j4="neon-attachments",O4=1,Qn="attachments";let Ps=null;async function ql(){return Ps||new Promise((r,t)=>{const n=indexedDB.open(j4,O4);n.onerror=()=>{var a;t(new Error(`无法打开附件数据库: ${(a=n.error)==null?void 0:a.message}`))},n.onsuccess=()=>{Ps=n.result,r(Ps)},n.onupgradeneeded=a=>{const i=a.target.result;i.objectStoreNames.contains(Qn)||i.createObjectStore(Qn,{keyPath:"id"}).createIndex("messageId","messageId",{unique:!1})}})}async function ex(r){if(r.length===0)return;const t=await ql();return new Promise((n,a)=>{const s=t.transaction(Qn,"readwrite").objectStore(Qn);let c=0,u=!1;for(const p of r){const{previewUrl:f,...h}=p,x=s.put(h);x.onsuccess=()=>{c++,c===r.length&&!u&&n()},x.onerror=()=>{var y;u||(u=!0,a(new Error(`保存附件失败: ${(y=x.error)==null?void 0:y.message}`)))}}})}async function tx(r){const t=await ql();return new Promise((n,a)=>{const u=t.transaction(Qn,"readonly").objectStore(Qn).index("messageId").getAll(r);u.onsuccess=()=>n(u.result||[]),u.onerror=()=>{var p;return a(new Error(`获取消息附件失败: ${(p=u.error)==null?void 0:p.message}`))}})}async function M4(r){const t=await tx(r);if(t.length===0)return;const n=await ql();return new Promise((a,i)=>{const c=n.transaction(Qn,"readwrite").objectStore(Qn);let u=0;for(const p of t){const f=c.delete(p.id);f.onsuccess=()=>{u++,u===t.length&&a()},f.onerror=()=>{u++,u===t.length&&a()}}})}const Am=Object.freeze(Object.defineProperty({__proto__:null,deleteAttachmentsByMessageId:M4,getAttachmentsByMessageId:tx,openAttachmentDB:ql,saveAttachments:ex},Symbol.toStringTag,{value:"Module"}));function rx(r){const n="."+r.name.toLowerCase().split(".").pop();return Ir.ACCEPTED_DOCUMENT_EXTENSIONS.includes(n)}async function $4(r){return new Promise((t,n)=>{const a=new FileReader;a.onload=()=>{const i=a.result;if(!i||!i.trim()){n(new Error("EMPTY_DOCUMENT"));return}t(i)},a.onerror=()=>{n(new Error("READ_ERROR"))},a.readAsText(r,"utf-8")})}function U4(r){const t=r.type.toLowerCase();return t==="text/plain"||t==="text/markdown"||t==="text/x-markdown"?!0:rx(r)}function z4(r){return Ir.ACCEPTED_IMAGE_FORMATS.includes(r.type)}function H4(r){return z4(r)?"image":U4(r)?"document":null}const Hn=pi;function V4(){const{t:r}=ft(),{messages:t,llmConfigs:n,activeConfigId:a,getActiveConfig:i,currentMotion:s,isGenerating:c,error:u,currentConversationId:p,conversationList:f,addMessage:h,setCurrentMotion:x,setIsGenerating:y,setError:w,clarifySession:E,isClarifying:C,clarifyEnabled:S,setClarifySession:P,setIsClarifying:b,answerClarifyQuestion:A,skipClarify:T,nextClarifyQuestion:B,renderError:F,isFixing:k,fixAttemptCount:N,setIsFixing:I,incrementFixAttempt:L,setFixAbortController:M,clearErrorState:H,touchConversation:z,updateConversationTitle:Z,saveCurrentConversation:oe,importConversations:re,addToast:se,pendingAttachments:Y,addPendingAttachment:te,updatePendingAttachment:ee,removePendingAttachment:_,clearPendingAttachments:$}=Ht(),G=n.length>0&&a!==null,K=G&&f.length===0&&t.length===0,[ue,me]=D.useState(""),Le=D.useRef(null),he=D.useCallback(()=>`temp_${Date.now()}_${Math.random().toString(36).substring(2,9)}`,[]),Ve=D.useCallback(async ke=>{const xe=Ir.MAX_ATTACHMENTS-Y.length;for(let ye=0;ye<Math.min(ke.length,xe);ye++){const Fe=ke[ye],He=he(),Oe=H4(Fe);if(!Oe){te({tempId:He,file:Fe,status:"error",error:Vu.INVALID_IMAGE_FORMAT});continue}const ct=Oe==="image"?URL.createObjectURL(Fe):void 0;te({tempId:He,file:Fe,status:"pending",previewUrl:ct}),ot(He,Fe,Oe)}},[Y.length,te,he]),ot=D.useCallback(async(ke,xe,ye)=>{ee(ke,{status:"processing"});try{if(ye==="image"){if(xe.size>Ir.MAX_IMAGE_SIZE)throw new Error("FILE_TOO_LARGE");if(!await iC(xe))throw new Error("INVALID_IMAGE_FORMAT");const He=await Ag(xe),{blob:Oe,wasCompressed:ct}=await sC(xe),rt=await lC(Oe),Rt={id:he(),messageId:"",type:"image",fileName:xe.name,mimeType:xe.type,content:rt,wasCompressed:ct,originalDimensions:He,createdAt:Date.now()};ee(ke,{status:"ready",attachment:Rt,wasCompressed:ct})}else{if(!rx(xe))throw new Error("INVALID_DOCUMENT_FORMAT");const Fe=await $4(xe),He={id:he(),messageId:"",type:"document",fileName:xe.name,mimeType:xe.type||"text/plain",content:Fe,createdAt:Date.now()};ee(ke,{status:"ready",attachment:He})}}catch(Fe){const He=Fe instanceof Error?Fe.message:"READ_ERROR",Oe=Vu[He]||Vu.READ_ERROR;ee(ke,{status:"error",error:Oe})}},[ee,he]);D.useEffect(()=>{var ke;(ke=Le.current)==null||ke.scrollIntoView({behavior:"smooth"})},[t]);const Vt=D.useCallback(ke=>{var Fe;const xe=(Fe=ke.clipboardData)==null?void 0:Fe.items;if(!xe)return;const ye=[];for(const He of xe)if(He.type.startsWith("image/")){const Oe=He.getAsFile();Oe&&ye.push(Oe)}if(ye.length>0){const He=new DataTransfer;ye.forEach(Oe=>He.items.add(Oe)),Ve(He.files)}},[Ve]),We=D.useCallback(async ke=>{y(!0);try{const xe=Array.isArray(ke);console.log("[ChatPanel] 开始调用 LLM 服务，使用提示词:",xe?"[多模态内容]":ke);const ye=await i();if(!ye)throw new Error(r("chat.configError"));const Fe={baseURL:ye.baseURL,apiKey:ye.apiKey,model:ye.model},He=Rm(Fe);let Oe;s?(console.log("[ChatPanel] 修改现有动效",xe?"[多模态内容]":""),Oe=await He.modifyMotion(ke,s,t)):(console.log("[ChatPanel] 生成新动效"),Oe=await He.generateMotion(ke)),console.log("[ChatPanel] 收到动效定义, 设置 currentMotion:",Oe),x(Oe);const ct={id:Hn(),role:"assistant",content:r(s?"chat.motionUpdated":"chat.motionGenerated"),timestamp:Date.now(),motionSnapshot:Oe.id};h(ct),z(),oe()}catch(xe){console.error("[ChatPanel] 生成动效失败:",xe);const ye=xe instanceof Error?xe.message:r("chat.generateError");w(ye);const Fe={id:Hn(),role:"assistant",content:r("chat.generateErrorDetail",{message:ye}),timestamp:Date.now()};h(Fe)}finally{y(!1),P(null),b(!1)}},[r,i,s,t,h,y,x,w,P,b,z,oe]),Ee=D.useCallback(async ke=>{if(!E||!G)return;const xe=E.questions[E.currentQuestionIndex];if(!xe)return;const ye=xe.options.find(rt=>rt.id===ke),Fe=ye?ye.label:ke,He={id:Hn(),role:"user",content:`${xe.question}
→ ${Fe}`,timestamp:Date.now()};h(He);const Oe={questionId:xe.id,selectedOptionId:ke,customValue:null};if(A(xe.id,Oe),E.currentQuestionIndex>=E.questions.length-1){const rt=await i();if(!rt)return;const Rt={baseURL:rt.baseURL,apiKey:rt.apiKey,model:rt.model},Ke=Ju(Rt),mt={...E,answers:[...E.answers,Oe],status:"completed"},Tt=Ke.buildFinalPrompt(mt);console.log("[ChatPanel] 澄清完成，最终提示词:",Tt);const Ft=E.attachments&&E.attachments.length>0?Va(Tt,E.attachments):Tt;await We(Ft)}else B()},[E,G,i,h,A,B,We]),ge=D.useCallback(async ke=>{if(!E||!G)return;const xe=E.questions[E.currentQuestionIndex];if(!xe)return;const ye={id:Hn(),role:"user",content:`${xe.question}
→ ${ke}`,timestamp:Date.now()};h(ye);const Fe={questionId:xe.id,selectedOptionId:null,customValue:ke};if(A(xe.id,Fe),E.currentQuestionIndex>=E.questions.length-1){const Oe=await i();if(!Oe)return;const ct={baseURL:Oe.baseURL,apiKey:Oe.apiKey,model:Oe.model},rt=Ju(ct),Rt={...E,answers:[...E.answers,Fe],status:"completed"},Ke=rt.buildFinalPrompt(Rt);console.log("[ChatPanel] 澄清完成（自定义答案），最终提示词:",Ke);const mt=E.attachments&&E.attachments.length>0?Va(Ke,E.attachments):Ke;await We(mt)}else B()},[E,G,i,h,A,B,We]),je=D.useCallback(async()=>{if(!E)return;T();const ke=E.attachments&&E.attachments.length>0?Va(E.originalPrompt,E.attachments):E.originalPrompt;await We(ke)},[E,T,We]),Ye=D.useCallback(async()=>{if(!s||!F||!G||k||N>=Fs.MAX_FIX_ATTEMPTS)return;console.log("[ChatPanel] 开始修复, 尝试次数:",N+1);const ke=new AbortController;M(ke),I(!0),L();try{const xe=await i();if(!xe)throw new Error("无法获取 LLM 配置");const ye={baseURL:xe.baseURL,apiKey:xe.apiKey,model:xe.model},He=await Rm(ye).fixMotion(s,F,{signal:ke.signal});console.log("[ChatPanel] 修复成功，更新动效"),x(He);const Oe={id:Hn(),role:"assistant",content:r("chat.fixSuccess"),timestamp:Date.now(),motionSnapshot:He.id};h(Oe)}catch(xe){if(xe.name==="AbortError"){console.log("[ChatPanel] 修复请求已取消");return}console.error("[ChatPanel] 修复失败:",xe);const ye=xe instanceof Error?xe.message:r("chat.fixError"),Fe={id:Hn(),role:"assistant",content:r("chat.fixErrorDetail",{message:ye}),timestamp:Date.now()};h(Fe)}finally{I(!1),M(null)}},[r,s,F,G,k,N,i,M,I,L,x,h]);D.useEffect(()=>{(F==null?void 0:F.source)==="postProcess"&&!k&&N<Fs.MAX_FIX_ATTEMPTS&&G&&s&&Ye()},[F==null?void 0:F.id]);const tt=D.useCallback(async()=>{var xe;const ke=f.map(ye=>ye.title);try{const ye=await fetch("/neon/preset-starter.neon");if(!ye.ok)throw new Error("Failed to load preset file");const Fe=await ye.text(),He=new Blob([Fe],{type:"application/json"}),Oe=new File([He],"preset-starter.neon",{type:"application/json"}),ct=await Jg(Oe,ke);ct.success&&ct.conversations?(re(ct.conversations),se({type:"success",message:r("chat.importSuccess")})):se({type:"error",message:((xe=ct.errors[0])==null?void 0:xe.message)||r("chat.importFailed")})}catch(ye){console.error("[ChatPanel] 导入预设对话失败:",ye),se({type:"error",message:r("chat.importError")})}},[r,f,re,se]),Dt=async ke=>{if(ke.preventDefault(),!ue.trim()||!G||c||C)return;const xe=ue.trim();me(""),w(null),H();const ye=Y.filter(Ke=>Ke.status==="ready"&&Ke.attachment).map(Ke=>Ke.attachment),Fe=await i();if(!Fe){w(r("chat.configError"));return}const He={baseURL:Fe.baseURL,apiKey:Fe.apiKey,model:Fe.model},Oe=Hn();if(ye.length>0){const Ke=ye.map(mt=>({...mt,messageId:Oe}));try{await ex(Ke)}catch(mt){console.error("[ChatPanel] 保存附件失败:",mt)}}const ct={id:Oe,role:"user",content:xe,timestamp:Date.now(),attachmentIds:ye.length>0?ye.map(Ke=>Ke.id):void 0};if(h(ct),Y.length>0&&$(),t.length===0&&p){const Ke=xe.length<=30?xe:xe.substring(0,30)+"...";Z(p,Ke)}z();let rt=xe;if(S&&!s){b(!0);try{console.log("[ChatPanel] 开始分析是否需要澄清");const Ke=Ju(He),mt=ye.length>0?Va(rt,ye):rt,Tt=await Ke.analyzePrompt(mt);if(Tt.needsClarification&&Tt.questions.length>0){const Ft={id:`clarify_${Date.now()}`,originalPrompt:xe,questions:Tt.questions,answers:[],currentQuestionIndex:0,status:"questioning",createdAt:Date.now(),updatedAt:Date.now(),attachments:ye.length>0?ye:void 0};P(Ft),console.log("[ChatPanel] 需要澄清，创建会话:",Ft);const cr={id:Hn(),role:"assistant",content:r("chat.clarifyIntro"),timestamp:Date.now()};h(cr);return}else rt=Tt.directPrompt||rt,console.log("[ChatPanel] 无需澄清，直接生成"),b(!1)}catch(Ke){console.error("[ChatPanel] 澄清分析失败，跳过澄清:",Ke),b(!1)}}const Rt=ye.length>0?Va(rt,ye):rt;await We(Rt)};return g.jsxs("div",{className:"flex flex-col h-full",children:[g.jsx(P4,{}),g.jsxs("div",{className:"flex-1 overflow-y-auto p-3 space-y-3",children:[t.length===0&&g.jsx("div",{className:"text-center text-text-muted py-8",children:K?g.jsxs(g.Fragment,{children:[g.jsx("p",{className:"mb-4 font-body text-text-primary text-base",children:r("chat.emptyTitle")}),g.jsx("p",{className:"mb-6 font-body text-sm text-text-secondary",children:r("chat.emptyDescription")}),g.jsx(lt,{variant:"secondary",size:"lg",onClick:tt,disabled:c,children:r("chat.importDemo")})]}):g.jsxs(g.Fragment,{children:[g.jsx("p",{className:"mb-2 font-body",children:r("chat.emptyFallbackTitle")}),g.jsx("p",{className:"text-sm font-body",children:r("chat.emptyFallbackHint")})]})}),t.map(ke=>g.jsx("div",{className:`flex ${ke.role==="user"?"justify-end":"justify-start"}`,children:g.jsxs("div",{className:`max-w-[85%] rounded-lg px-3 py-2 text-sm font-body ${ke.role==="user"?"bg-accent-primary text-black":"bg-background-elevated text-text-primary border border-border-default"}`,children:[ke.content,ke.role==="user"&&ke.attachmentIds&&ke.attachmentIds.length>0&&g.jsx(o4,{messageId:ke.id,attachmentIds:ke.attachmentIds})]})},ke.id)),C&&!E&&g.jsx("div",{className:"flex justify-start",children:g.jsx("div",{className:"bg-background-elevated rounded-lg px-3 py-2 text-sm font-body text-text-muted border border-border-default",children:g.jsxs("span",{className:"inline-flex items-center gap-1",children:[g.jsx("span",{className:"animate-pulse",children:r("chat.analyzing")}),g.jsx("span",{className:"animate-bounce",children:"..."})]})})}),c&&g.jsx("div",{className:"flex justify-start",children:g.jsx("div",{className:"bg-background-elevated rounded-lg px-3 py-2 text-sm font-body text-text-muted border border-border-default",children:g.jsxs("span",{className:"inline-flex items-center gap-1",children:[g.jsx("span",{className:"animate-pulse",children:r("chat.generating")}),g.jsx("span",{className:"animate-bounce",children:"..."})]})})}),E&&E.status==="questioning"&&g.jsx("div",{className:"flex justify-start",children:g.jsx("div",{className:"w-full max-w-[90%] bg-background-elevated rounded-lg px-3 py-3 border border-border-default",children:g.jsx(Zb,{question:E.questions[E.currentQuestionIndex],progress:{current:E.currentQuestionIndex+1,total:E.questions.length},onSelectOption:Ee,onSubmitCustom:ge,onSkip:je,disabled:c})})}),F&&g.jsx("div",{className:"flex justify-start",children:g.jsx("div",{className:"w-full max-w-[90%]",children:g.jsx(e4,{error:F,onFix:Ye,loading:k,disabled:k||c,attemptCount:N,maxAttempts:Fs.MAX_FIX_ATTEMPTS})})}),g.jsx("div",{ref:Le})]}),u&&g.jsx("div",{className:"px-3 py-2 bg-accent-tertiary/10 border-t border-accent-tertiary/20",children:g.jsx("p",{className:"text-sm font-body text-accent-tertiary",children:u})}),Y.length>0&&g.jsx(n4,{attachments:Y,onRemove:_,disabled:c}),g.jsx("form",{onSubmit:Dt,className:"p-3 border-t border-border-default",children:g.jsxs("div",{className:"flex gap-2 items-center",children:[g.jsx(t4,{onFileSelect:Ve,disabled:c||C||!G,currentCount:Y.length}),g.jsx(wo,{value:ue,onChange:ke=>me(ke.target.value),onPaste:Vt,placeholder:r(C?"chat.placeholder3":s?"chat.placeholder2":"chat.placeholder1"),disabled:c||C||!G,className:"flex-1"}),g.jsx(lt,{type:"submit",variant:"primary",size:"md",disabled:!ue.trim()||c||C||!G,loading:c,children:r("chat.send")})]})})]})}function W4(){const{t:r}=ft(),{aspectRatio:t,setAspectRatio:n}=Ht(),a=s=>{n(s)},i=Vl(t);return g.jsxs("div",{className:"flex flex-col gap-2",children:[g.jsxs("div",{className:"flex items-center justify-between",children:[g.jsx("span",{className:"text-xs font-body text-text-muted",children:r("aspectRatio.label")}),g.jsx("span",{className:"text-xs font-medium font-body text-text-primary",children:r(i.label)})]}),g.jsx("div",{className:"flex flex-wrap gap-1",children:Dg.map(s=>g.jsx("button",{onClick:()=>a(s.id),className:`
              px-2 py-1 text-xs rounded-md border transition-colors cursor-pointer
              ${t===s.id?"bg-accent-primary text-black border-accent-primary shadow-neon-soft":"bg-background-elevated text-text-muted border-border-default hover:border-accent-primary hover:text-accent-primary"}
            `,title:r(s.label),children:s.id},s.id))})]})}function G4({config:r,isActive:t,onSelect:n,onEdit:a,onDelete:i,disabled:s=!1}){const{t:c}=ft(),p=(f=>{try{const h=new URL(f).hostname;return h.includes("openai")?"OpenAI":h.includes("deepseek")?"DeepSeek":h.includes("anthropic")?"Anthropic":h.includes("azure")?"Azure":h.split(".")[0]}catch{return"API"}})(r.baseURL);return g.jsx("div",{className:`
        p-3 rounded-lg border transition-colors cursor-pointer
        ${t?"border-accent-primary bg-accent-primary/10 shadow-neon-soft":"border-border-default hover:border-accent-secondary/50 hover:bg-background-secondary"}
        ${s?"opacity-50 pointer-events-none":""}
      `,onClick:n,children:g.jsxs("div",{className:"flex items-start justify-between gap-3",children:[g.jsxs("div",{className:"flex-1 min-w-0",children:[g.jsxs("div",{className:"flex items-center gap-2",children:[t&&g.jsx("span",{className:"w-2 h-2 rounded-full bg-accent-primary flex-shrink-0 shadow-neon-soft"}),g.jsx("h4",{className:"text-sm font-medium font-body text-text-primary truncate",children:r.name})]}),g.jsxs("div",{className:"mt-1 flex items-center gap-2 text-xs font-body text-text-muted",children:[g.jsx("span",{className:"px-1.5 py-0.5 rounded bg-background-elevated border border-border-default truncate",children:p}),g.jsx("span",{className:"truncate",children:r.model})]})]}),g.jsxs("div",{className:"flex items-center gap-1 flex-shrink-0",onClick:f=>f.stopPropagation(),children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:a,disabled:s,className:"!p-1.5",title:c("config.edit"),children:g.jsx("svg",{className:"w-4 h-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"})})}),g.jsx(lt,{variant:"ghost",size:"sm",onClick:i,disabled:s,className:"!p-1.5 text-accent-tertiary hover:text-accent-tertiary hover:bg-accent-tertiary/10",title:c("config.delete"),children:g.jsx("svg",{className:"w-4 h-4",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"})})})]})]})})}const q4="gpt-4";function Pm({initialData:r,onSave:t,onCancel:n,isSaving:a=!1,isEditing:i=!1}){const{t:s}=ft(),[c,u]=D.useState((r==null?void 0:r.name)||""),[p,f]=D.useState((r==null?void 0:r.baseURL)||y2),[h,x]=D.useState((r==null?void 0:r.apiKey)||""),[y,w]=D.useState((r==null?void 0:r.model)||q4),[E,C]=D.useState({});D.useEffect(()=>{r&&(u(r.name),f(r.baseURL),x(r.apiKey),w(r.model))},[r]);const S=()=>{const b={};if(c.trim()||(b.name=s("config.form.nameRequired")),!p.trim())b.baseURL=s("config.form.baseURLRequired");else try{const A=new URL(p);["http:","https:"].includes(A.protocol)||(b.baseURL=s("config.form.baseURLInvalidProtocol"))}catch{b.baseURL=s("config.form.baseURLInvalid")}return h.trim()||(b.apiKey=s("config.form.apiKeyRequired")),y.trim()||(b.model=s("config.form.modelRequired")),C(b),Object.keys(b).length===0},P=()=>{S()&&t({name:c.trim(),baseURL:p.trim(),apiKey:h.trim(),model:y.trim()})};return g.jsxs("div",{className:"space-y-4",children:[g.jsx(wo,{label:s("config.form.name"),value:c,onChange:b=>u(b.target.value),error:E.name,placeholder:s("config.form.namePlaceholder"),helperText:s("config.form.nameHelper")}),g.jsx(wo,{label:s("config.form.baseURL"),value:p,onChange:b=>f(b.target.value),error:E.baseURL,placeholder:"https://api.openai.com/v1",helperText:s("config.form.baseURLHelper")}),g.jsx(wo,{label:"API Key",type:"password",value:h,onChange:b=>x(b.target.value),error:E.apiKey,placeholder:"sk-...",helperText:i?s("config.form.apiKeyEditHelper"):void 0}),g.jsx(wo,{label:s("config.form.model"),value:y,onChange:b=>w(b.target.value),error:E.model,placeholder:s("config.form.modelPlaceholder"),helperText:s("config.form.modelHelper")}),g.jsxs("div",{className:"flex justify-end gap-3 pt-4",children:[g.jsx(lt,{variant:"ghost",onClick:n,disabled:a,children:s("common.cancel")}),g.jsx(lt,{variant:"primary",onClick:P,loading:a,children:s(i?"config.form.save":"config.add")})]})]})}function K4(){const{t:r}=ft(),{llmConfigs:t,activeConfigId:n,isLoadingConfigs:a,addLLMConfig:i,updateLLMConfig:s,deleteLLMConfig:c,setActiveConfigId:u}=Ht(),[p,f]=D.useState("list"),[h,x]=D.useState(null),[y,w]=D.useState(null),[E,C]=D.useState(!1),[S,P]=D.useState(null),b=()=>{const M=Ul(),H=$l(M);return M||Sg(Cg(H)),H},A=async M=>{C(!0);try{const H=b(),z=await oi(M.apiKey,H),Z=new Date().toISOString(),oe={id:pi(),name:M.name,baseURL:M.baseURL,apiKey:z,model:M.model,createdAt:Z,updatedAt:Z};i(oe),f("list")}catch(H){console.error("[ConfigList] Failed to add config:",H)}finally{C(!1)}},T=async M=>{try{const H=b(),z=await Id(M.apiKey,H);x(M),w({name:M.name,baseURL:M.baseURL,apiKey:z,model:M.model}),f("edit")}catch(H){console.error("[ConfigList] Failed to decrypt config for editing:",H),alert(r("config.decryptFailed"))}},B=async M=>{if(h){C(!0);try{const H=b();let z;M.apiKey&&M.apiKey!==(y==null?void 0:y.apiKey)?z=await oi(M.apiKey,H):z=h.apiKey,s(h.id,{name:M.name,baseURL:M.baseURL,apiKey:z,model:M.model}),f("list"),x(null),w(null)}catch(H){console.error("[ConfigList] Failed to update config:",H)}finally{C(!1)}}},F=()=>{S&&(c(S.id),P(null))},k=()=>{f("list"),x(null),w(null)},N=()=>g.jsxs("div",{className:"space-y-3",children:[g.jsxs("div",{className:"flex items-center justify-between",children:[g.jsx("h3",{className:"text-sm font-medium font-body text-text-primary",children:r("config.title")}),g.jsxs(lt,{variant:"secondary",size:"sm",onClick:()=>f("add"),disabled:a,children:[g.jsx("svg",{className:"w-4 h-4 mr-1",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M12 4v16m8-8H4"})}),r("config.add")]})]}),a?g.jsx("div",{className:"py-8 text-center text-sm font-body text-text-muted",children:r("config.loading")}):t.length===0?g.jsx("div",{className:"py-8 text-center",children:g.jsx("p",{className:"text-sm font-body text-text-muted",children:r("config.empty")})}):g.jsx("div",{className:"space-y-2",children:t.map(M=>g.jsx(G4,{config:M,isActive:M.id===n,onSelect:()=>u(M.id),onEdit:()=>T(M),onDelete:()=>P(M),disabled:a},M.id))}),t.length>0&&g.jsx("p",{className:"text-xs font-body text-text-muted mt-2",children:r("config.switchHint")})]}),I=()=>g.jsxs("div",{className:"space-y-3",children:[g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:k,className:"!p-1",children:g.jsx("svg",{className:"w-5 h-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M15 19l-7-7 7-7"})})}),g.jsx("h3",{className:"text-sm font-medium font-body text-text-primary",children:r("config.add")})]}),g.jsx(Pm,{onSave:A,onCancel:k,isSaving:E})]}),L=()=>g.jsxs("div",{className:"space-y-3",children:[g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:k,className:"!p-1",children:g.jsx("svg",{className:"w-5 h-5",fill:"none",viewBox:"0 0 24 24",stroke:"currentColor",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M15 19l-7-7 7-7"})})}),g.jsx("h3",{className:"text-sm font-medium font-body text-text-primary",children:r("config.edit")})]}),y&&g.jsx(Pm,{initialData:y,onSave:B,onCancel:k,isSaving:E,isEditing:!0})]});return g.jsxs(g.Fragment,{children:[p==="list"&&N(),p==="add"&&I(),p==="edit"&&L(),g.jsx(Ng,{isOpen:S!==null,title:r("config.delete"),message:r("config.deleteConfirmMessage",{name:S==null?void 0:S.name}),confirmLabel:r("config.delete"),cancelLabel:r("common.cancel"),variant:"danger",onConfirm:F,onCancel:()=>P(null)})]})}function X4({isOpen:r,onClose:t}){const{t:n}=ft(),{clarifyEnabled:a,setClarifyEnabled:i}=Ht();return r?g.jsxs("div",{className:"fixed inset-0 z-50 flex items-center justify-center",children:[g.jsx("div",{className:"absolute inset-0 bg-black/60 backdrop-blur-sm",onClick:t}),g.jsx("div",{className:"relative bg-background-elevated rounded-lg shadow-neon-soft w-full max-w-md mx-4 overflow-hidden max-h-[90vh] flex flex-col border border-border-default",children:g.jsxs("div",{className:"p-6 overflow-y-auto flex-1",children:[g.jsx("h2",{className:"text-lg font-semibold font-display text-text-primary mb-4",children:n("settings.title")}),g.jsxs("div",{className:"space-y-4",children:[g.jsx(K4,{}),g.jsxs("div",{className:"pt-4 border-t border-border-default",children:[g.jsx(Od,{label:n("settings.clarify.label"),checked:a,onChange:s=>i(s.target.checked)}),g.jsx("p",{className:"mt-1 text-xs font-body text-text-muted",children:n("settings.clarify.description")})]}),g.jsx("div",{className:"pt-4 border-t border-border-default",children:g.jsxs("div",{className:"flex items-center justify-between",children:[g.jsxs("div",{children:[g.jsx("p",{className:"text-sm font-medium font-body text-text-primary",children:n("settings.log.title")}),g.jsx("p",{className:"mt-1 text-xs font-body text-text-muted",children:n("settings.log.description")})]}),g.jsx(lt,{variant:"secondary",size:"sm",onClick:()=>pe.exportToFile(),children:n("settings.log.export")})]})})]}),g.jsx("div",{className:"mt-6 flex justify-end",children:g.jsx(lt,{variant:"primary",onClick:t,children:n("settings.close")})})]})})]}):null}function nx(){const r={canvas:typeof HTMLCanvasElement<"u"};return r.canvas?{supported:!0,capabilities:r}:{supported:!1,reason:"您的浏览器不支持 Canvas",capabilities:r}}class Y4{async prepareForExport(t,n={}){const{fps:a=60,cloneVideos:i=!0}=n;pe.info("FrameByFrameRenderer","准备导出环境",{fps:a,cloneVideos:i});const s=t.getDimensions(),c=document.createElement("canvas");c.width=s.width,c.height=s.height;let u={};if(i){const p=t.getParameters();u=await Wn.cloneAllVideosForExport(p),pe.debug("FrameByFrameRenderer","视频克隆完成",{count:Object.keys(u).length})}return{offscreenCanvas:c,clonedVideos:u,renderer:t,fps:a}}async renderFrame(t,n,a={}){const{offscreenCanvas:i,clonedVideos:s,renderer:c}=t,u={...a,videoOverrides:{...a.videoOverrides,...s},waitForVideoSeek:!0};await c.renderToCanvas(i,n,u);const p=i.getContext("2d");if(!p)throw new Error("无法获取离屏 canvas context");return p.getImageData(0,0,i.width,i.height)}async*renderAllFrames(t,n,a){const i=1e3/n,s=Math.max(1,Math.ceil(a/i));pe.info("FrameByFrameRenderer","开始逐帧渲染",{fps:n,duration:a,totalFrames:s});for(let c=0;c<s;c++){const u=c*i,p=await this.renderFrame(t,u);yield{frameIndex:c,time:u,imageData:p},c%10===0&&await new Promise(f=>setTimeout(f,0))}pe.info("FrameByFrameRenderer","逐帧渲染完成",{totalFrames:s})}calculateTotalFrames(t,n){const a=1e3/t;return Math.max(1,Math.ceil(n/a))}cleanup(t){Wn.disposeClonedVideos(t.clonedVideos);const n=t.offscreenCanvas.getContext("2d");n&&n.clearRect(0,0,t.offscreenCanvas.width,t.offscreenCanvas.height),pe.debug("FrameByFrameRenderer","导出上下文已清理")}}const Vr=class Vr{constructor(){ve(this,"encoder",null);ve(this,"_isInitialized",!1);ve(this,"isLoading",!1);ve(this,"alignedWidth",0);ve(this,"alignedHeight",0)}async initialize(t,n,a={}){if(this._isInitialized){pe.warn("H264EncoderService","编码器已初始化，请先调用 dispose()");return}if(this.isLoading){pe.warn("H264EncoderService","编码器正在加载中");return}this.isLoading=!0;try{if(typeof window.__initH264MP4Encoder__!="function")throw new Error("H264 MP4 编码器未正确加载，请确保 h264-mp4-encoder 库已加载");this.encoder=await window.__initH264MP4Encoder__(),this.alignedWidth=t%2===0?t:t+1,this.alignedHeight=n%2===0?n:n+1,this.encoder.width=this.alignedWidth,this.encoder.height=this.alignedHeight,this.encoder.frameRate=a.frameRate??Vr.DEFAULT_FRAME_RATE,this.encoder.quantizationParameter=a.quality??Vr.DEFAULT_QUALITY,this.encoder.speed=a.speed??Vr.DEFAULT_SPEED,this.encoder.groupOfPictures=a.keyframeInterval??Vr.DEFAULT_KEYFRAME_INTERVAL,this.encoder.initialize(),this._isInitialized=!0,pe.info("H264EncoderService","编码器初始化成功",{width:this.alignedWidth,height:this.alignedHeight,frameRate:this.encoder.frameRate,quality:this.encoder.quantizationParameter})}catch(i){throw pe.error("H264EncoderService","编码器初始化失败",{error:i instanceof Error?i.message:String(i)}),i}finally{this.isLoading=!1}}addFrame(t){if(!this._isInitialized||!this.encoder)throw new Error("编码器未初始化，请先调用 initialize()");(t.width!==this.alignedWidth||t.height!==this.alignedHeight)&&pe.warn("H264EncoderService","图像尺寸与编码器不匹配",{expected:{width:this.alignedWidth,height:this.alignedHeight},actual:{width:t.width,height:t.height}}),this.encoder.addFrameRgba(t.data)}addFrameRgba(t){if(!this._isInitialized||!this.encoder)throw new Error("编码器未初始化，请先调用 initialize()");this.encoder.addFrameRgba(t)}finalize(){if(!this._isInitialized||!this.encoder)throw new Error("编码器未初始化，请先调用 initialize()");this.encoder.finalize();const t=this.encoder.FS.readFile(this.encoder.outputFilename),n=new Blob([t],{type:"video/mp4"});return pe.info("H264EncoderService","编码完成",{size:n.size,sizeKB:(n.size/1024).toFixed(1)}),n}dispose(){if(this.encoder){try{this.encoder.delete(),pe.debug("H264EncoderService","编码器资源已释放")}catch(t){pe.warn("H264EncoderService","释放编码器资源时出错",{error:t instanceof Error?t.message:String(t)})}this.encoder=null}this._isInitialized=!1,this.alignedWidth=0,this.alignedHeight=0}isInitialized(){return this._isInitialized}getAlignedWidth(){return this.alignedWidth}getAlignedHeight(){return this.alignedHeight}static isEncoderAvailable(){return typeof window.__initH264MP4Encoder__=="function"}static alignDimension(t){return t%2===0?t:t+1}};ve(Vr,"DEFAULT_FRAME_RATE",60),ve(Vr,"DEFAULT_QUALITY",28),ve(Vr,"DEFAULT_SPEED",5),ve(Vr,"DEFAULT_KEYFRAME_INTERVAL",60);let si=Vr;class ks extends Error{constructor(n,a,i,s){super(n);ve(this,"type");ve(this,"friendlyMessage");ve(this,"phase");this.name="ExportError",this.type=a,this.friendlyMessage=i,this.phase=s}toRenderError(n){return{id:`error_${Date.now()}_${Math.random().toString(36).substr(2,9)}`,type:this.type,message:this.message,friendlyMessage:this.friendlyMessage,code:n,timestamp:Date.now()}}}const ed=100;class Q4{constructor(){ve(this,"frameRenderer");ve(this,"encoder",null);ve(this,"status","idle");ve(this,"cancelled",!1);ve(this,"currentContext",null);ve(this,"lastError",null);this.frameRenderer=new Y4}validateExport(t,n){const a=[],i=t.getDuration();return i<ed&&a.push({code:"DURATION_TOO_SHORT",message:`动效时长 ${i}ms 小于最小要求 ${ed}ms`,friendlyMessage:`动效时长太短，请确保时长至少为 ${ed/1e3} 秒`}),(n.frameRate<1||n.frameRate>120)&&a.push({code:"INVALID_FRAME_RATE",message:`帧率 ${n.frameRate} 超出有效范围 (1-120)`,friendlyMessage:"帧率设置无效，请使用 1-120 之间的值"}),["720p","1080p","4k"].includes(n.resolution)||a.push({code:"INVALID_RESOLUTION",message:`分辨率 ${n.resolution} 不在支持列表中`,friendlyMessage:"请选择有效的分辨率（720p、1080p 或 4K）"}),si.isEncoderAvailable()||a.push({code:"ENCODER_UNAVAILABLE",message:"H.264 MP4 编码器未加载",friendlyMessage:"MP4 编码器未能加载，请刷新页面重试"}),{valid:a.length===0,errors:a}}getLastError(){return this.lastError}generateFriendlyMessage(t,n){const a=t instanceof Error?t.message:String(t);return n==="preparing"?a.includes("视频")?"准备视频资源时出错，请检查视频文件是否有效":"准备导出环境时出错，请稍后重试":n==="rendering"?a.includes("canvas")?"渲染画布时出错，请检查动效代码是否正确":"渲染帧时出错，请检查动效代码是否有运行时错误":n==="encoding"?a.includes("内存")||a.includes("memory")?"编码时内存不足，请尝试降低分辨率或帧率":"视频编码时出错，请尝试重新导出":a==="导出已取消"?"导出已取消":"导出失败，请稍后重试"}async exportToMP4(t,n,a){if(this.status!=="idle")throw new ks("导出流水线正在运行中，请先取消当前导出","export","导出正在进行中，请等待当前导出完成或取消后重试",this.status);this.lastError=null,this.cancelled=!1,this.status="preparing",pe.info("ExportPipeline","开始导出",{config:n});const i=this.validateExport(t,n);if(!i.valid){const s=i.errors[0],c=new ks(s.message,"export",s.friendlyMessage,"preparing");throw this.lastError=c,this.status="error",c}try{const s=n.aspectRatio||Hl,c=Wl(s,n.resolution),u=t.getDuration(),p=this.frameRenderer.calculateTotalFrames(n.frameRate,u);if(a==null||a(0),this.currentContext=await this.frameRenderer.prepareForExport(t,{fps:n.frameRate,cloneVideos:!0}),this.cancelled)throw new Error("导出已取消");if(this.encoder=new si,await this.encoder.initialize(c.width,c.height,{frameRate:n.frameRate,quality:10,speed:5,keyframeInterval:n.frameRate}),this.cancelled)throw new Error("导出已取消");this.status="rendering";const f=document.createElement("canvas");f.width=this.encoder.getAlignedWidth(),f.height=this.encoder.getAlignedHeight();const h=f.getContext("2d");if(!h)throw new Error("无法创建导出 canvas context");let x=0;for await(const w of this.frameRenderer.renderAllFrames(this.currentContext,n.frameRate,u)){if(this.cancelled)throw new Error("导出已取消");const E=document.createElement("canvas");E.width=w.imageData.width,E.height=w.imageData.height;const C=E.getContext("2d");C&&(C.putImageData(w.imageData,0,0),h.drawImage(E,0,0,f.width,f.height));const S=h.getImageData(0,0,f.width,f.height);this.encoder.addFrame(S),x++;const P=Math.min(x/p*90+5,95);a==null||a(P)}if(this.cancelled)throw new Error("导出已取消");this.status="encoding",a==null||a(96);const y=this.encoder.finalize();return a==null||a(100),this.status="complete",pe.info("ExportPipeline","导出完成",{size:y.size,sizeKB:(y.size/1024).toFixed(1),frames:x}),y}catch(s){if(pe.error("ExportPipeline","导出失败",{error:s instanceof Error?s.message:String(s),phase:this.status}),s instanceof Error&&s.message==="导出已取消")throw this.status="idle",s;if(s instanceof ks)throw this.lastError=s,this.status="error",s;const c=this.status==="encoding"?"encode":"export",u=this.generateFriendlyMessage(s,this.status),p=new ks(s instanceof Error?s.message:String(s),c,u,this.status);throw this.lastError=p,this.status="error",p}finally{this.cleanup()}}cancel(){this.status!=="idle"&&(pe.info("ExportPipeline","取消导出"),this.cancelled=!0,this.cleanup())}isExporting(){return this.status!=="idle"&&this.status!=="complete"}getStatus(){return this.status}cleanup(){this.currentContext&&(this.frameRenderer.cleanup(this.currentContext),this.currentContext=null),this.encoder&&(this.encoder.dispose(),this.encoder=null),this.status="idle"}}async function J4(){if(si.isEncoderAvailable())return;const r=document.createElement("script");r.src="/neon/h264mp4/h264-mp4-encoder.web.js",await new Promise((t,n)=>{r.onload=()=>{typeof window.HME<"u"?(window.__initH264MP4Encoder__=()=>window.HME.createH264MP4Encoder(),pe.info("Exporter","H.264 编码器加载成功"),t()):n(new Error("H.264 编码器加载失败：HME 未定义"))},r.onerror=()=>n(new Error("H.264 编码器脚本加载失败")),document.head.appendChild(r)})}class Z4{constructor(){ve(this,"renderer",null);ve(this,"container",null);ve(this,"exportPipeline",null)}async export(t,n,a){await J4();const i=n.aspectRatio||Hl,s=n.resolution||"1080p",c=Wl(i,s);this.container=document.createElement("div"),this.container.style.cssText=`
      position: fixed;
      left: -9999px;
      top: -9999px;
      width: ${c.width}px;
      height: ${c.height}px;
      visibility: hidden;
      pointer-events: none;
    `,document.body.appendChild(this.container);const u={...t,width:c.width,height:c.height};pe.info("Exporter","使用 ExportPipeline 导出",{resolution:s,frameRate:n.frameRate,dimensions:c}),this.renderer=Gg(u,{exportMode:!0}),await this.renderer.initialize(this.container,u),await new Promise(p=>setTimeout(p,50));try{return this.exportPipeline=new Q4,await this.exportPipeline.exportToMP4(this.renderer,{...n,aspectRatio:i,resolution:s},a)}finally{this.cleanup()}}cleanup(){this.renderer&&(this.renderer.destroy(),this.renderer=null),this.container&&(this.container.remove(),this.container=null),this.exportPipeline&&(this.exportPipeline.cancel(),this.exportPipeline=null)}cancel(){this.cleanup()}isSupported(){const t=nx();return{supported:t.supported,reason:t.reason}}download(t,n){const a=URL.createObjectURL(t),i=document.createElement("a");i.href=a,i.download=n,document.body.appendChild(i),i.click(),document.body.removeChild(i),URL.revokeObjectURL(a)}}const td=new Z4;function eS(r){return`motion_${new Date().toISOString().slice(0,19).replace(/[:-]/g,"")}.mp4`}const tS=[{label:"720p",value:"720p"},{label:"1080p",value:"1080p"},{label:"4K",value:"4k"}],rS=[{label:"24 fps",value:"24"},{label:"30 fps",value:"30"},{label:"60 fps",value:"60"}];function nS({isOpen:r,onClose:t}){const{t:n}=ft(),{currentMotion:a,isExporting:i,exportProgress:s,setIsExporting:c,setExportProgress:u,setError:p,aspectRatio:f}=Ht(),[h,x]=D.useState("1080p"),[y,w]=D.useState(30),[E,C]=D.useState(null),S=D.useMemo(()=>Wl(f,h),[f,h]);if(D.useEffect(()=>{r&&C(nx())},[r]),!r)return null;const P=async()=>{if(a){c(!0),u(0),p(null);try{const A={resolution:h,frameRate:y,format:"webm",aspectRatio:f},T=await td.export(a,A,F=>u(F)),B=eS(a);td.download(T,B),t()}catch(A){const T=A instanceof Error?A.message:n("export.failed");T!=="Export cancelled"&&p(T)}finally{c(!1),u(0)}}},b=()=>{i&&td.cancel(),t()};return g.jsxs("div",{className:"fixed inset-0 z-50 flex items-center justify-center",children:[g.jsx("div",{className:"absolute inset-0 bg-black/60 backdrop-blur-sm",onClick:i?void 0:b}),g.jsx("div",{className:"relative bg-background-elevated rounded-lg shadow-neon-soft w-full max-w-md mx-4 overflow-hidden border border-border-default",children:g.jsxs("div",{className:"p-6",children:[g.jsx("h2",{className:"text-lg font-semibold font-display text-text-primary mb-4",children:n("export.videoTitle")}),E&&!E.supported?g.jsx("div",{className:"p-4 bg-accent-tertiary/10 rounded-lg text-accent-tertiary mb-4 border border-accent-tertiary/30",children:E.reason}):i?g.jsxs("div",{className:"space-y-4",children:[g.jsx("div",{className:"text-center text-text-muted mb-2",children:n("export.exporting")}),g.jsx("div",{className:"w-full bg-border-default rounded-full h-3",children:g.jsx("div",{className:"bg-accent-primary h-3 rounded-full transition-all duration-150",style:{width:`${s}%`}})}),g.jsxs("div",{className:"text-center text-sm text-text-muted",children:[Math.round(s),"%"]})]}):g.jsxs("div",{className:"space-y-4",children:[g.jsx(kl,{label:n("export.resolution"),options:tS,value:h,onChange:A=>x(A.target.value)}),g.jsx(kl,{label:n("export.frameRate"),options:rS,value:String(y),onChange:A=>w(Number(A.target.value))}),g.jsxs("div",{className:"text-sm text-text-muted bg-background-secondary p-3 rounded-lg border border-border-default",children:[g.jsx("p",{className:"font-body",children:n("export.duration",{duration:a?(a.duration/1e3).toFixed(1):"0"})}),g.jsx("p",{className:"mt-1 font-body",children:n("export.outputSize",{width:S.width,height:S.height})}),g.jsx("p",{className:"mt-1 font-body",children:n("export.outputFormatMp4")}),h==="4k"&&g.jsx("p",{className:"mt-2 text-accent-secondary font-body",children:n("export.4kWarning")})]})]}),g.jsxs("div",{className:"mt-6 flex justify-end gap-3",children:[g.jsx(lt,{variant:"ghost",onClick:b,children:n(i?"common.cancel":"export.close")}),!i&&(E==null?void 0:E.supported)&&g.jsx(lt,{variant:"primary",onClick:P,disabled:!a,children:n("export.startExport")})]})]})})]})}function oS(r){return{number:"🔢",color:"🎨",boolean:"🔘",select:"📋",image:"🖼️"}[r]||"📌"}const aS=({parameters:r,onToggle:t,onSelectAll:n,onDeselectAll:a})=>{const{t:i}=ft(),s=w=>{const E=`paramSelector.type.${w}`,C=i(E);return C!==E?C:w},c=w=>{var S;const{parameter:E,currentValue:C}=w;switch(E.type){case"number":{const P=C;return E.unit?`${P}${E.unit}`:String(P)}case"color":return C;case"boolean":return i(C?"paramSelector.booleanOn":"paramSelector.booleanOff");case"select":{const P=(S=E.options)==null?void 0:S.find(b=>b.value===C);return(P==null?void 0:P.label)||C}case"image":return i("paramSelector.imageValue");default:return String(C)}},u=r.filter(w=>w.exportable),p=u.filter(w=>w.selected).length,f=u.length,h=p===f&&f>0,x=p===0,y=r.filter(w=>w.exportable);return r.length===0?g.jsx("div",{className:"parameter-selector-empty",children:g.jsx("p",{className:"text-gray-400 text-sm",children:i("paramSelector.empty")})}):g.jsxs("div",{className:"parameter-selector",children:[g.jsxs("div",{className:"flex items-center justify-between mb-3",children:[g.jsx("span",{className:"text-sm text-gray-400",children:i("paramSelector.selectedCount",{selected:p,total:f})}),g.jsxs("div",{className:"flex gap-2",children:[g.jsx("button",{type:"button",onClick:n,disabled:h,className:`text-xs px-2 py-1 rounded transition-colors ${h?"text-gray-500 cursor-not-allowed":"text-cyan-400 hover:bg-cyan-400/10"}`,children:i("paramSelector.selectAll")}),g.jsx("button",{type:"button",onClick:a,disabled:x,className:`text-xs px-2 py-1 rounded transition-colors ${x?"text-gray-500 cursor-not-allowed":"text-cyan-400 hover:bg-cyan-400/10"}`,children:i("paramSelector.deselectAll")})]})]}),g.jsx("div",{className:"space-y-2 max-h-64 overflow-y-auto pr-1",children:y.map(w=>g.jsxs("label",{className:`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all ${w.selected?"bg-cyan-500/10 border-cyan-500/30":"bg-gray-800/50 border-gray-700 hover:border-gray-600"}`,children:[g.jsx("input",{type:"checkbox",checked:w.selected,onChange:()=>t(w.parameter.id),className:"w-4 h-4 accent-cyan-500 cursor-pointer"}),g.jsxs("div",{className:"flex-1 min-w-0",children:[g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx("span",{className:"text-base",title:s(w.parameter.type),children:oS(w.parameter.type)}),g.jsx("span",{className:"font-medium text-gray-200 truncate",children:w.parameter.name})]}),g.jsxs("div",{className:"flex items-center gap-2 mt-1 text-xs text-gray-500",children:[g.jsx("span",{className:"px-1.5 py-0.5 bg-gray-700/50 rounded",children:s(w.parameter.type)}),g.jsx("span",{className:"truncate",children:i("paramSelector.currentValue",{value:c(w)})})]})]})]},w.parameter.id))})]})};function Ya(r){return!r.parameters||r.parameters.length===0?[]:r.parameters.map(t=>({parameter:t,selected:km(),path:t.path,currentValue:iS(t),exportable:km()}))}function km(r){return!0}function iS(r){var t,n;switch(r.type){case"number":return r.value??r.min??0;case"color":return r.colorValue??"#000000";case"boolean":return r.boolValue??!1;case"select":return r.selectedValue??((n=(t=r.options)==null?void 0:t[0])==null?void 0:n.value)??"";case"image":return r.imageValue??r.placeholderImage??"";case"video":return r.videoValue??r.placeholderVideo??"";case"string":return r.stringValue??"";default:return""}}function sS(r){switch(r){case"number":return"slider";case"color":return"color";case"boolean":return"toggle";case"select":return"select";case"image":return"image";case"video":return"video";case"string":return"text";default:return"slider"}}function lS(r){return r.filter(t=>t.selected&&t.exportable).map(t=>{const n=t.parameter,a={id:n.id,label:n.name,controlType:sS(n.type),initialValue:t.currentValue,path:t.path};return n.type==="number"&&(a.numberConfig={min:n.min??0,max:n.max??100,step:n.step??1,unit:n.unit}),n.type==="select"&&n.options&&(a.selectConfig={options:n.options.map(i=>({value:i.value,label:i.label}))}),a})}function rd(r,t){return r.map(n=>({...n,selected:n.exportable&&t.includes(n.parameter.id)}))}function Dm(r){return r.filter(t=>t.selected&&t.exportable).map(t=>t.parameter.id)}function cS(r){return r.map(t=>({...t,selected:t.exportable}))}function uS(r){return r.map(t=>({...t,selected:!1}))}function dS(r,t){return r.map(n=>n.parameter.id===t&&n.exportable?{...n,selected:!n.selected}:n)}const _l=class _l{generateVideoSyncService(){return`
  // ============================================
  // 视频同步服务 (与主平台 VideoSyncService 保持一致)
  // ============================================

  var VideoSyncService = {
    DEFAULT_TIMEOUT: ${_l.DEFAULT_VIDEO_SYNC_TIMEOUT},

    /**
     * 同步单个视频到指定时间
     * @param video - 视频元素
     * @param timeMs - 目标时间（毫秒）
     * @param options - { timeout, loop, startTimeOffset }
     */
    syncVideoToTime: function(video, timeMs, options) {
      options = options || {};
      var timeout = options.timeout !== undefined ? options.timeout : this.DEFAULT_TIMEOUT;
      var loop = options.loop !== undefined ? options.loop : true;
      var startTimeOffset = options.startTimeOffset || 0;

      // 计算有效时间：动效时间 - 视频起始偏移 (024-video-start-time)
      var effectiveTimeMs = timeMs - startTimeOffset;
      var timeInSeconds = Math.max(0, effectiveTimeMs) / 1000;
      var videoDuration = video.duration || 1;

      var targetTime;
      if (loop) {
        targetTime = timeInSeconds % videoDuration;
        // 边界处理：避免跳回第一帧
        if (targetTime === 0 && timeInSeconds > 0) {
          targetTime = videoDuration - 0.001;
        }
      } else {
        // 非循环模式：超出时长后保持最后一帧
        targetTime = Math.min(timeInSeconds, videoDuration - 0.001);
      }

      // 如果已经在目标位置，直接返回
      if (Math.abs(video.currentTime - targetTime) < 0.01) {
        return Promise.resolve();
      }

      // 确保视频暂停
      if (!video.paused) {
        video.pause();
      }

      return new Promise(function(resolve) {
        var onSeeked = function() {
          video.removeEventListener('seeked', onSeeked);
          resolve();
        };

        video.addEventListener('seeked', onSeeked);
        video.currentTime = targetTime;

        // 超时保护
        setTimeout(function() {
          video.removeEventListener('seeked', onSeeked);
          resolve();
        }, timeout);
      });
    },

    /**
     * 同步所有视频参数到指定时间
     * @param params - 参数对象
     * @param timeMs - 目标时间（毫秒）
     * @param options - 同步选项
     * @param startTimeOffsets - 各视频的起始时间偏移映射 (024-video-start-time)
     */
    syncAllVideosToTime: function(params, timeMs, options, startTimeOffsets) {
      var self = this;
      var promises = [];
      startTimeOffsets = startTimeOffsets || {};

      for (var paramId in params) {
        var value = params[paramId];
        // 检查是否是 HTMLVideoElement
        if (value && value.tagName === 'VIDEO' && typeof value.currentTime !== 'undefined') {
          var syncOptions = Object.assign({}, options, {
            startTimeOffset: startTimeOffsets[paramId] || 0
          });
          promises.push(self.syncVideoToTime(value, timeMs, syncOptions));
        }
      }

      if (promises.length === 0) {
        return Promise.resolve();
      }

      return Promise.all(promises);
    }
  };
`.trim()}generateParameterLoader(t){return`
  // ============================================
  // 参数加载器 (与主平台 ParameterLoaderService 保持一致)
  // ============================================

  var ParameterLoader = {
    /**
     * 加载参数初始值
     */
    loadInitialParams: function() {
      return ${this.generateParamsInitCode(t)};
    },

    /**
     * 预加载图片参数
     * @param params - 参数对象
     * @param imageAssets - 图片资源映射 { paramId: base64/url }
     */
    preloadImages: function(params, imageAssets) {
      var promises = [];

      Object.keys(imageAssets).forEach(function(paramId) {
        var promise = new Promise(function(resolve) {
          var img = new Image();
          img.onload = function() {
            params[paramId] = img;
            resolve();
          };
          img.onerror = function() {
            console.warn('图片加载失败:', paramId);
            resolve(); // 失败也继续
          };
          img.src = imageAssets[paramId];
        });
        promises.push(promise);
      });

      return Promise.all(promises);
    },

    /**
     * 预加载视频参数
     * @param params - 参数对象
     * @param videoAssets - 视频资源映射 { paramId: blobUrl }
     */
    preloadVideos: function(params, videoAssets) {
      var promises = [];

      Object.keys(videoAssets).forEach(function(paramId) {
        var promise = new Promise(function(resolve) {
          var video = document.createElement('video');
          video.muted = true;
          video.playsInline = true;
          video.preload = 'auto';

          video.onloadeddata = function() {
            params[paramId] = video;
            resolve();
          };
          video.onerror = function() {
            console.warn('视频加载失败:', paramId);
            resolve();
          };
          video.src = videoAssets[paramId];
        });
        promises.push(promise);
      });

      return Promise.all(promises);
    }
  };
`.trim()}generateRenderingCore(t){const{motion:n,overrideDimensions:a}=t,i=(a==null?void 0:a.width)??n.width,s=(a==null?void 0:a.height)??n.height,c=n.code||"",u=!!(n.postProcessCode&&n.postProcessCode.trim()),p=u?n.postProcessCode.trim():"";return`
  // ============================================
  // 渲染核心 (与主平台 CanvasRenderer 保持一致)
  // ============================================

  var RenderingCore = {
    canvas: null,
    ctx: null,
    renderFunction: null,
    canvasInfo: null,
    ${u?`
    // 后处理模式专用
    offscreenCanvas: null,
    offscreenCtx: null,
    postProcessor: null,
    postProcessFn: null,`:""}

    /**
     * 初始化渲染核心
     */
    initialize: function() {
      this.canvas = document.getElementById('motion-canvas');
      ${u?`
      // 后处理模式：隐藏原始 canvas，使用离屏渲染
      this.canvas.style.display = 'none';
      this.offscreenCanvas = document.createElement('canvas');
      this.offscreenCanvas.width = ${i};
      this.offscreenCanvas.height = ${s};
      this.offscreenCtx = this.offscreenCanvas.getContext('2d');`:`
      this.ctx = this.canvas.getContext('2d');`}

      // 设置 Canvas 尺寸（像素尺寸）
      this.canvas.width = ${i};
      this.canvas.height = ${s};
      // 显示尺寸由 CSS 的 max-width/max-height 控制

      this.canvasInfo = {
        width: ${i},
        height: ${s}
      };

      // 加载渲染函数
      this.loadRenderFunction();

      ${u?`
      // 初始化后处理器
      if (typeof PostProcessRuntime !== 'undefined') {
        this.postProcessor = new PostProcessRuntime();
        this.postProcessor.initialize(this.canvas.parentElement, ${i}, ${s});

        // 加载后处理函数
        this.loadPostProcessFunction();
      }`:""}
    },

    /**
     * 加载用户定义的渲染函数
     */
    loadRenderFunction: function() {
      try {
        // 用户定义的渲染代码
        ${c}

        // 获取渲染函数
        if (typeof window.__motionRender === 'function') {
          this.renderFunction = window.__motionRender;
          window.__motionRender = undefined;
        }
      } catch (e) {
        console.error('加载渲染函数失败:', e);
      }
    },

    ${u?`
    /**
     * 加载后处理函数
     */
    loadPostProcessFunction: function() {
      try {
        // 用户定义的后处理代码
        ${p}

        // 获取后处理函数
        if (typeof window.__motionPostProcess === 'function') {
          this.postProcessFn = window.__motionPostProcess;
          window.__motionPostProcess = undefined;
        }
      } catch (e) {
        console.error('加载后处理函数失败:', e);
      }
    },`:""}

    /**
     * 渲染单帧
     * @param ctx - Canvas 上下文
     * @param time - 当前时间（毫秒）
     * @param params - 参数对象
     * @param transparent - 是否透明背景
     */
    renderFrame: function(ctx, time, params, transparent) {
      var width = this.canvasInfo.width;
      var height = this.canvasInfo.height;

      if (transparent) {
        // 透明模式：清除画布
        ctx.clearRect(0, 0, width, height);
      } else {
        // 普通模式：填充背景色
        ctx.fillStyle = '${this.escapeString(n.backgroundColor||"#ffffff")}';
        ctx.fillRect(0, 0, width, height);
      }

      // 调用渲染函数
      if (this.renderFunction) {
        try {
          ctx.save();
          this.renderFunction(ctx, time, params, this.canvasInfo);
          ctx.restore();
        } catch (e) {
          console.error('渲染错误:', e);
        }
      }
    },

    ${u?`
    /**
     * 渲染单帧（后处理模式）
     * 先渲染到离屏 canvas，再通过后处理器输出到主 canvas
     * @param time - 当前时间（毫秒）
     * @param params - 参数对象
     * @param transparent - 是否透明背景
     */
    renderFrameWithPostProcess: function(time, params, transparent) {
      // 1. 渲染到离屏 canvas
      this.renderFrame(this.offscreenCtx, time, params, transparent);

      // 2. 应用后处理
      if (this.postProcessor && this.postProcessFn) {
        this.postProcessor.render(this.offscreenCanvas, time, params, this.postProcessFn);
      }
    },`:""}

    /**
     * 渲染到离屏 Canvas（用于导出隔离）
     * @param targetCanvas - 离屏 canvas
     * @param time - 渲染时间
     * @param params - 参数对象
     * @param transparent - 是否透明
     */
    renderToOffscreen: function(targetCanvas, time, params, transparent) {
      var targetCtx = targetCanvas.getContext('2d');
      this.renderFrame(targetCtx, time, params, transparent);
    }
  };
`.trim()}generateAnimationLoop(t){const{motion:n}=t,a=!!(n.postProcessCode&&n.postProcessCode.trim());return`
  // ============================================
  // 动画循环 (与主平台 CanvasRenderer 保持一致)
  // ============================================

  var AnimationLoop = {
    isPlaying: true,
    startTime: 0,
    pausedTime: 0,
    animationFrameId: null,
    duration: ${n.duration},
    effectiveDuration: 0, // 动态计算的有效时长 (025-dynamic-duration)
    params: null,

    /**
     * 初始化动画循环
     * @param params - 参数对象
     */
    initialize: function(params) {
      this.params = params;
      this.startTime = performance.now();
      // 计算初始动态时长 (025-dynamic-duration)
      this.updateDynamicDuration();
    },

    /**
     * 更新动态时长 (025-dynamic-duration)
     */
    updateDynamicDuration: function() {
      var motionLike = {
        duration: this.duration,
        durationCode: motionDurationCode,
        parameters: parametersDefinition
      };
      var previousDuration = this.effectiveDuration;
      this.effectiveDuration = VideoStartTimeEvaluator.getEffectiveDuration(motionLike, this.params);
      if (previousDuration !== this.effectiveDuration && previousDuration !== 0) {
        console.log('[025-dynamic-duration] 时长已更新:', previousDuration, '->', this.effectiveDuration);
      }
    },

    /**
     * 获取有效时长 (025-dynamic-duration)
     */
    getEffectiveDuration: function() {
      return this.effectiveDuration || this.duration;
    },

    /**
     * 动画帧回调
     */
    tick: async function() {
      if (!this.isPlaying) return;

      var currentTime = performance.now() - this.startTime;

      // 循环动画（使用动态时长 025-dynamic-duration）
      var loopDuration = this.effectiveDuration || this.duration;
      if (currentTime >= loopDuration) {
        this.startTime = performance.now();
        currentTime = 0;
      }

      // 精确定位所有视频参数（传递起始时间偏移）(024-video-start-time)
      await VideoSyncService.syncAllVideosToTime(this.params, currentTime, {}, videoStartTimeOffsets);

      // T037: 更新序列帧参数到当前帧 (029-sequence-frame-input)
      SequenceService.updateSequenceFrames(this.params, currentTime);

      // 渲染当前帧
      ${a?`
      RenderingCore.renderFrameWithPostProcess(currentTime, this.params, false);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, currentTime, this.params, false);`}

      var self = this;
      this.animationFrameId = requestAnimationFrame(function() {
        self.tick();
      });
    },

    /**
     * 开始播放
     */
    play: function() {
      if (this.isPlaying) return;
      this.isPlaying = true;
      this.startTime = performance.now() - this.pausedTime;
      this.tick();
    },

    /**
     * 暂停
     */
    pause: function() {
      if (!this.isPlaying) return;
      this.isPlaying = false;
      this.pausedTime = performance.now() - this.startTime;
      if (this.animationFrameId) {
        cancelAnimationFrame(this.animationFrameId);
        this.animationFrameId = null;
      }
    },

    /**
     * 停止
     */
    stop: function() {
      this.pause();
      this.pausedTime = 0;
      ${a?`
      RenderingCore.renderFrameWithPostProcess(0, this.params, false);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, 0, this.params, false);`}
    },

    /**
     * 跳转到指定时间
     * @param time - 目标时间（毫秒）
     */
    seek: async function(time) {
      this.pausedTime = time;
      this.startTime = performance.now() - time;
      // 传递起始时间偏移 (024-video-start-time)
      await VideoSyncService.syncAllVideosToTime(this.params, time, {}, videoStartTimeOffsets);
      // T037: 更新序列帧参数到当前帧 (029-sequence-frame-input)
      SequenceService.updateSequenceFrames(this.params, time);
      ${a?`
      RenderingCore.renderFrameWithPostProcess(time, this.params, false);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, time, this.params, false);`}
    },

    /**
     * 获取当前时间
     */
    getCurrentTime: function() {
      if (this.isPlaying) {
        return performance.now() - this.startTime;
      }
      return this.pausedTime;
    }
  };
`.trim()}generateExportInterface(t){const{motion:n}=t,a=!!(n.postProcessCode&&n.postProcessCode.trim());return`
  // ============================================
  // 导出接口 (motionControls)
  // ============================================

  window.motionControls = {
    // 导出状态管理
    _isExporting: false,
    _preExportTime: 0,

    play: function() {
      AnimationLoop.play();
    },

    pause: function() {
      AnimationLoop.pause();
    },

    stop: function() {
      AnimationLoop.stop();
    },

    seek: async function(time) {
      await AnimationLoop.seek(time);
    },

    seekTo: function(time) {
      this.seek(time);
    },

    setProgress: function(progress) {
      // 使用动态时长 (025-dynamic-duration)
      var time = progress * AnimationLoop.getEffectiveDuration();
      this.seek(time);
    },

    isPlaying: function() {
      return AnimationLoop.isPlaying;
    },

    getDuration: function() {
      // 返回动态时长 (025-dynamic-duration)
      return AnimationLoop.getEffectiveDuration();
    },

    getCurrentTime: function() {
      return AnimationLoop.getCurrentTime();
    },

    /**
     * 开始导出模式
     * 保存当前预览状态，防止导出过程影响预览
     */
    beginExport: function() {
      this._isExporting = true;
      this._preExportTime = AnimationLoop.getCurrentTime();
      // 暂停动画循环
      AnimationLoop.pause();
    },

    /**
     * 结束导出模式
     * 恢复预览到导出前的状态
     */
    endExport: async function() {
      this._isExporting = false;
      // 恢复到导出前的时间点并渲染
      await VideoSyncService.syncAllVideosToTime(params, this._preExportTime, {}, videoStartTimeOffsets);
      ${a?`
      RenderingCore.renderFrameWithPostProcess(this._preExportTime, params, false);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, this._preExportTime, params, false);`}
    },

    /**
     * 渲染指定时间的帧（用于导出）
     * @param time - 时间（毫秒）
     * @param transparent - 是否透明背景
     */
    renderAt: async function(time, transparent) {
      // 传递起始时间偏移 (024-video-start-time)
      await VideoSyncService.syncAllVideosToTime(params, time, {}, videoStartTimeOffsets);
      ${a?`
      RenderingCore.renderFrameWithPostProcess(time, params, transparent);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, time, params, transparent);`}
    },

    /**
     * 渲染到离屏 Canvas（用于导出隔离）
     * @param targetCanvas - 离屏 canvas
     * @param time - 时间（毫秒）
     * @param transparent - 是否透明
     * @param videoOverrides - 克隆的视频对象映射
     */
    renderToCanvas: async function(targetCanvas, time, transparent, videoOverrides) {
      // 使用克隆的视频对象（如果有）
      var renderParams = {};
      for (var paramId in params) {
        if (videoOverrides && videoOverrides[paramId]) {
          renderParams[paramId] = videoOverrides[paramId];
        } else {
          renderParams[paramId] = params[paramId];
        }
      }

      // 同步视频到指定时间（传递起始时间偏移）(024-video-start-time)
      await VideoSyncService.syncAllVideosToTime(renderParams, time, {}, videoStartTimeOffsets);

      ${a?`
      // 后处理模式：先渲染到离屏，再应用后处理，最后复制到目标
      if (RenderingCore.postProcessor) {
        RenderingCore.renderToOffscreen(RenderingCore.offscreenCanvas, time, renderParams, transparent);
        RenderingCore.postProcessor.render(RenderingCore.offscreenCanvas, time, renderParams, RenderingCore.postProcessFn);
        var targetCtx = targetCanvas.getContext('2d');
        targetCtx.drawImage(RenderingCore.postProcessor.getCanvas(), 0, 0);
      } else {
        RenderingCore.renderToOffscreen(targetCanvas, time, renderParams, transparent);
      }`:`
      // 标准路径：直接渲染到目标 canvas
      RenderingCore.renderToOffscreen(targetCanvas, time, renderParams, transparent);`}
    }
  };

  // 参数更新函数
  window.updateParam = function(paramId, value) {
    params[paramId] = value;

    // 参数变更可能影响 videoStartTimeCode 的动态计算结果 (024-video-start-time)
    // 例如：videoStartTimeCode 引用了其他数值参数
    updateVideoStartTimeOffsets();

    // 参数变更可能影响 durationCode 的动态计算结果 (025-dynamic-duration)
    AnimationLoop.updateDynamicDuration();

    if (!AnimationLoop.isPlaying) {
      ${a?`
      RenderingCore.renderFrameWithPostProcess(AnimationLoop.pausedTime, params, false);`:`
      RenderingCore.renderFrame(RenderingCore.ctx, AnimationLoop.pausedTime, params, false);`}
    }
  };

  // 获取参数值
  window.getParam = function(paramId) {
    return params[paramId];
  };

  // 暴露 params 对象
  window.__motionParams = params;

  // 更新动效时长
  window.updateMotionDuration = function(newDuration) {
    AnimationLoop.duration = newDuration;
    console.log('动效时长已更新:', newDuration, 'ms');
  };
`.trim()}generateVideoStartTimeEvaluator(){return`
  // ============================================
  // 视频起始时间评估器 (024-video-start-time)
  // 复用自 videoStartTimeEvaluator.ts
  // ============================================

  ${Vg}
`.trim()}generateSequenceService(){return`
  // ============================================
  // 序列帧服务 (T035-T036, 029-sequence-frame-input)
  // ============================================

  var SequenceService = {
    // 预加载的序列帧缓存: { paramId: HTMLImageElement[] }
    preloadedSequences: {},

    /**
     * T035: 预加载所有序列帧
     * 从 __SEQUENCE_ASSETS__ 中读取 Base64 数据并转换为 HTMLImageElement
     */
    preloadSequences: function() {
      var self = this;
      var assets = window.__SEQUENCE_ASSETS__ || {};

      Object.keys(assets).forEach(function(paramId) {
        var assetData = assets[paramId];
        if (!assetData || !assetData.frames || assetData.frames.length === 0) {
          return;
        }

        var loadedFrames = [];
        assetData.frames.forEach(function(base64Data) {
          var img = new Image();
          img.src = base64Data;
          loadedFrames.push(img);
        });

        self.preloadedSequences[paramId] = loadedFrames;
        console.log('[SequenceService] 预加载完成:', paramId, '帧数:', loadedFrames.length);
      });
    },

    /**
     * T036: 根据时间获取当前序列帧
     * @param paramId - 参数 ID
     * @param time - 当前时间（毫秒）
     * @param fps - 帧率（默认 30）
     * @param loop - 是否循环（默认 true）
     * @returns HTMLImageElement 或 null
     */
    getSequenceFrame: function(paramId, time, fps, loop) {
      var frames = this.preloadedSequences[paramId];
      if (!frames || frames.length === 0) {
        return null;
      }

      fps = fps || 30;
      loop = loop !== undefined ? loop : true;

      var frameDuration = 1000 / fps;
      var frameIndex = Math.floor(time / frameDuration);

      if (loop) {
        return frames[frameIndex % frames.length];
      } else {
        return frames[Math.min(frameIndex, frames.length - 1)];
      }
    },

    /**
     * 获取序列帧的 fps 和 loop 配置
     * @param paramId - 参数 ID
     */
    getSequenceConfig: function(paramId) {
      var assets = window.__SEQUENCE_ASSETS__ || {};
      var assetData = assets[paramId];
      if (!assetData) {
        return { fps: 30, loop: true };
      }
      return {
        fps: assetData.fps || 30,
        loop: assetData.loop !== undefined ? assetData.loop : true
      };
    },

    /**
     * T037: 更新所有序列帧参数到当前帧
     * 在每帧渲染前调用
     * @param params - 参数对象
     * @param time - 当前时间（毫秒）
     */
    updateSequenceFrames: function(params, time) {
      var self = this;
      Object.keys(this.preloadedSequences).forEach(function(paramId) {
        var config = self.getSequenceConfig(paramId);
        var currentFrame = self.getSequenceFrame(paramId, time, config.fps, config.loop);
        if (currentFrame) {
          params[paramId] = currentFrame;
        }
      });
    }
  };

  // 暴露到 window 以便控件绑定代码可以访问 (029-sequence-frame-input)
  window.SequenceService = SequenceService;
`.trim()}generateParametersDefinition(t){const n=t.map(a=>{const i={id:a.id,type:a.type};switch(a.type){case"number":i.value=a.value??0;break;case"color":i.colorValue=a.colorValue??"#000000";break;case"boolean":i.boolValue=a.boolValue??!1;break;case"select":i.selectedValue=a.selectedValue??"";break;case"video":i.videoStartTime=a.videoStartTime??0,i.videoStartTimeCode=a.videoStartTimeCode??null;break}return i});return JSON.stringify(n,null,2)}generateCompleteRendererCode(t){var h,x;const n=this.generateVideoSyncService(),a=this.generateVideoStartTimeEvaluator(),i=this.generateSequenceService(),s=this.generateParameterLoader(t.exportedParams),c=this.generateRenderingCore(t),u=this.generateAnimationLoop(t),p=this.generateExportInterface(t),f=this.generateParametersDefinition(t.motion.parameters);return`
// ============================================
// Canvas 渲染器 (统一渲染架构 023-unified-renderer)
// ============================================

(function() {
  'use strict';

  // 动效配置
  var config = {
    duration: ${t.motion.duration},
    width: ${((h=t.overrideDimensions)==null?void 0:h.width)??t.motion.width},
    height: ${((x=t.overrideDimensions)==null?void 0:x.height)??t.motion.height},
    backgroundColor: '${this.escapeString(t.motion.backgroundColor||"#ffffff")}'
  };

  // 动态时长计算代码 (025-dynamic-duration)
  var motionDurationCode = ${t.motion.durationCode?JSON.stringify(t.motion.durationCode):"null"};

  // 参数定义（用于视频起始时间和动态时长计算）(024-video-start-time, 025-dynamic-duration)
  var parametersDefinition = ${f};

${n}

${a}

${i}

${s}

  // 参数值（在 ParameterLoader 定义后初始化）
  var params = ParameterLoader.loadInitialParams();

  // 视频起始时间偏移缓存 (024-video-start-time)
  var videoStartTimeOffsets = {};

  // 更新视频起始时间偏移（视频加载后调用）
  function updateVideoStartTimeOffsets() {
    videoStartTimeOffsets = VideoStartTimeEvaluator.getAllVideoStartTimes(parametersDefinition, params);
    if (Object.keys(videoStartTimeOffsets).length > 0) {
      console.log('[024-video-start-time] 视频起始时间偏移已计算:', videoStartTimeOffsets);
    }
  }

${c}

${u}

${p}

  // 初始化
  RenderingCore.initialize();
  SequenceService.preloadSequences();  // T035: 预加载序列帧
  AnimationLoop.initialize(params);
  AnimationLoop.tick();
})();
`.trim()}generateAllSegments(t){return{videoSyncService:this.generateVideoSyncService(),parameterLoader:this.generateParameterLoader(t.exportedParams),renderingCore:this.generateRenderingCore(t),animationLoop:this.generateAnimationLoop(t),exportInterface:this.generateExportInterface(t)}}generateParamsInitCode(t){const n={};for(const a of t){const i=a.parameter;switch(i.type){case"number":case"color":case"boolean":case"select":case"image":case"string":n[i.id]=a.currentValue;break}}return JSON.stringify(n,null,2)}escapeString(t){return t?t.replace(/\\/g,"\\\\").replace(/'/g,"\\'").replace(/"/g,'\\"').replace(/\n/g,"\\n").replace(/\r/g,"\\r"):""}};ve(_l,"DEFAULT_VIDEO_SYNC_TIMEOUT",200);let vd=_l;const fS=new vd;function pS(r,t,n){const a={motion:r,exportedParams:t,overrideDimensions:n};return fS.generateCompleteRendererCode(a)}function hS(r){return Object.keys(r).length===0?"// 无图片资源需要预加载":`
// ============================================
// 图片资源预加载
// ============================================

(function() {
  // 图片预加载函数，在 window.updateParam 定义后调用
  function preloadImages() {
    var imageData = {
${Object.entries(r).map(([n,a])=>`    '${n}': '${a}'`).join(`,
`)}
  };

    var loadedImages = {};
    var loadPromises = [];

    Object.keys(imageData).forEach(function(paramId) {
      var promise = new Promise(function(resolve, reject) {
        var img = new Image();
        img.onload = function() {
          loadedImages[paramId] = img;
          resolve();
        };
        img.onerror = function() {
          console.warn('图片加载失败:', paramId);
          resolve(); // 即使失败也继续
        };
        img.src = imageData[paramId];
      });
      loadPromises.push(promise);
    });

    // 等待所有图片加载完成后更新参数
    Promise.all(loadPromises).then(function() {
      Object.keys(loadedImages).forEach(function(paramId) {
        if (window.updateParam) {
          window.updateParam(paramId, loadedImages[paramId]);
        } else {
          console.error('[ImagePreload] window.updateParam 未定义，无法更新图片参数:', paramId);
        }
      });
      console.log('[ImagePreload] 图片预加载完成，共', Object.keys(loadedImages).length, '张');
    });
  }

  // 通过 __onThreeReady 确保在渲染器代码（window.updateParam）之后执行
  if (window.__onThreeReady) {
    window.__onThreeReady(preloadImages);
  } else {
    // 降级处理：如果 __onThreeReady 不存在，直接执行
    console.warn('[ImagePreload] __onThreeReady 不存在，直接执行预加载');
    preloadImages();
  }
})();
`.trim()}function mS(r){return Object.keys(r).length===0?"// 无视频资源需要预加载":`
// ============================================
// 视频资源预加载 (019-video-input-support)
// ============================================

(function() {
  // 视频预加载函数，在 window.updateParam 定义后调用
  function preloadVideos() {
    var videoData = {
${Object.entries(r).map(([n,a])=>`    '${n}': '${a}'`).join(`,
`)}
  };

    var loadPromises = [];

    Object.keys(videoData).forEach(function(paramId) {
      var promise = new Promise(function(resolve) {
        var video = document.createElement('video');
        video.muted = true;
        video.playsInline = true;
        video.preload = 'auto';

        video.onloadeddata = function() {
          if (window.updateParam) {
            window.updateParam(paramId, video);
          } else {
            console.error('[VideoPreload] window.updateParam 未定义，无法更新视频参数:', paramId);
          }
          resolve();
        };

        video.onerror = function() {
          console.warn('[VideoPreload] 视频加载失败:', paramId);
          resolve();
        };

        video.src = videoData[paramId];
      });
      loadPromises.push(promise);
    });

    Promise.all(loadPromises).then(function() {
      console.log('[VideoPreload] 所有视频资源加载完成，共', loadPromises.length, '个');
    });
  }

  // 通过 __onThreeReady 确保在渲染器代码（window.updateParam）之后执行
  if (window.__onThreeReady) {
    window.__onThreeReady(preloadVideos);
  } else {
    // 降级处理：如果 __onThreeReady 不存在，直接执行
    console.warn('[VideoPreload] __onThreeReady 不存在，直接执行预加载');
    preloadVideos();
  }
})();
`.trim()}function gS(){return`
// ============================================
// Motion Utils - 确定性随机数工具
// ============================================

(function() {
  /**
   * 创建基于种子的随机数生成器 (Mulberry32 算法)
   * @param {number} seed - 种子值
   * @returns {function(): number} 返回 0-1 之间的随机数的函数
   */
  function createSeededRandom(seed) {
    var state = seed >>> 0;
    return function random() {
      state = (state + 0x6d2b79f5) >>> 0;
      var t = state;
      t = Math.imul(t ^ (t >>> 15), t | 1);
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  /**
   * 生成基于种子的随机整数
   * @param {number} seed - 种子值
   * @param {number} min - 最小值（包含）
   * @param {number} max - 最大值（包含）
   * @returns {number} 指定范围内的随机整数
   */
  function seededRandomInt(seed, min, max) {
    var random = createSeededRandom(seed);
    return Math.floor(random() * (max - min + 1)) + min;
  }

  /**
   * 生成基于种子的随机浮点数
   * @param {number} seed - 种子值
   * @param {number} min - 最小值（包含）
   * @param {number} max - 最大值（不包含）
   * @returns {number} 指定范围内的随机浮点数
   */
  function seededRandomRange(seed, min, max) {
    var random = createSeededRandom(seed);
    return random() * (max - min) + min;
  }

  /**
   * 创建基于种子和计数器的随机数生成器
   * @param {number} seed - 基础种子值
   * @returns {function(): number} 每次调用返回下一个随机数
   */
  function createRandomSequence(seed) {
    return createSeededRandom(seed);
  }

  // 全局注入 motion utils
  window.__motionUtils = {
    seededRandom: createSeededRandom,
    seededRandomInt: seededRandomInt,
    seededRandomRange: seededRandomRange,
    createRandomSequence: createRandomSequence
  };
})();
`.trim()}async function xS(r){return r.startsWith("data:")?r:new Promise((t,n)=>{const a=new Image;a.crossOrigin="anonymous",a.onload=()=>{try{const i=document.createElement("canvas");i.width=a.naturalWidth,i.height=a.naturalHeight;const s=i.getContext("2d");if(!s){n(new Error("Failed to get canvas context"));return}s.drawImage(a,0,0);const c=i.toDataURL("image/png");t(c)}catch(i){n(i)}},a.onerror=()=>{n(new Error(`Failed to load image: ${r}`))},a.src=r})}async function ox(r){const t={},a=r.filter(i=>i.parameter.type==="image").map(async i=>{const s=i.currentValue;if(!(!s||s==="__PLACEHOLDER__"))try{const c=await xS(s);t[i.parameter.id]=c}catch(c){console.warn(`Failed to encode image for parameter ${i.parameter.id}:`,c)}});return await Promise.all(a),t}function vS(r){let t=0;for(const n of Object.values(r))t+=n.length;return t}async function yS(r){return new Promise((t,n)=>{const a=new XMLHttpRequest;a.onload=()=>{const i=new FileReader;i.onloadend=()=>{t(i.result)},i.onerror=()=>{n(new Error("FileReader failed to convert blob to base64"))},i.readAsDataURL(a.response)},a.onerror=()=>{n(new Error(`Failed to fetch blob URL: ${r}`))},a.open("GET",r),a.responseType="blob",a.send()})}async function ax(r){var s;const t=r.filter(c=>c.type==="video");if(t.length===0)return pe.debug("videoEncoder","没有视频参数需要收集"),{assets:[],totalSize:0,errors:[]};pe.info("videoEncoder","开始收集视频资源",{count:t.length});const n=[],a=[];let i=0;for(const c of t){const u=c.videoValue;if(!(!u||u==="__PLACEHOLDER__"))try{if(!u.startsWith("blob:")){a.push({paramId:c.id,reason:"invalid_url",message:`视频 "${c.name}" 不是有效的 Blob URL`,url:u}),pe.warn("videoEncoder",`跳过无效 URL: ${c.id}`,{url:u});continue}const p=await yS(u);if(!p.startsWith("data:video/mp4")){a.push({paramId:c.id,reason:"unsupported_format",message:`视频 "${c.name}" 格式不支持，仅支持 MP4 格式`,url:u}),pe.warn("videoEncoder",`不支持的视频格式: ${c.id}`,{format:(s=p.split(";")[0])==null?void 0:s.split(":")[1]});continue}const f=Math.floor(p.length*.75);n.push({paramId:c.id,base64Data:p,format:"mp4",originalSize:f}),i+=p.length,pe.debug("videoEncoder",`视频已收集: ${c.id}`,{size:`${(f/1024/1024).toFixed(2)}MB`})}catch(p){a.push({paramId:c.id,reason:"fetch_failed",message:`视频 "${c.name}" 加载失败: ${p instanceof Error?p.message:String(p)}`,url:u}),pe.warn("videoEncoder",`收集视频失败: ${c.id}`,{error:p instanceof Error?p.message:String(p)})}}return pe.info("videoEncoder","视频资源收集完成",{successCount:n.length,errorCount:a.length,totalSize:`${(i/1024/1024).toFixed(2)}MB`}),{assets:n,totalSize:i,errors:a}}const ne={background:"#0a0a0f",backgroundSecondary:"#12121a",panelBackground:"rgba(20, 20, 30, 0.85)",panelBorder:"rgba(0, 212, 255, 0.3)",primary:"#00d4ff",primaryGlow:"rgba(0, 212, 255, 0.4)",textPrimary:"#e0e0e0",textSecondary:"#a0a0a0",textMuted:"#606060",borderRadius:"8px",shadow:"0 0 20px rgba(0, 212, 255, 0.2)",glassBlur:"10px"};function wS(){return`
/* ============================================ */
/* 科技风格主题 - Neon Motion Platform */
/* ============================================ */

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background: ${ne.background};
  color: ${ne.textPrimary};
  min-height: 100vh;
  overflow-x: hidden;
}

/* 应用容器 */
.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* 头部 */
.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: ${ne.panelBackground};
  backdrop-filter: blur(${ne.glassBlur});
  border-bottom: 1px solid ${ne.panelBorder};
}

.app-title {
  font-size: 20px;
  font-weight: 600;
  color: ${ne.primary};
  text-shadow: 0 0 10px ${ne.primaryGlow};
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-divider {
  width: 1px;
  height: 24px;
  background: ${ne.panelBorder};
  margin: 0 8px;
}

.control-btn {
  width: 40px;
  height: 40px;
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  background: ${ne.backgroundSecondary};
  color: ${ne.primary};
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.control-btn:hover {
  background: rgba(0, 212, 255, 0.1);
  border-color: ${ne.primary};
  box-shadow: ${ne.shadow};
}

.control-btn .icon {
  width: 20px;
  height: 20px;
}

/* 导出组 */
.export-group {
  display: flex;
  align-items: center;
  gap: 0;
}

/* 导出按钮 */
.export-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  background: ${ne.backgroundSecondary};
  color: ${ne.primary};
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.export-btn:hover:not(:disabled) {
  background: rgba(0, 212, 255, 0.1);
  border-color: ${ne.primary};
  box-shadow: ${ne.shadow};
}

.export-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.export-btn .icon {
  width: 18px;
  height: 18px;
}

.export-btn .icon.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.export-status {
  font-size: 12px;
  color: ${ne.primary};
  margin-left: 8px;
  display: none;
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  padding: 24px;
  gap: 24px;
}

/* Canvas 预览区 */
.preview-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

/* T011-T013: 背景图容器 (033-preview-background) */
.background-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: ${ne.borderRadius};
  overflow: hidden;
}

/* T011: 背景图控制按钮 (033-preview-background) */
.background-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
}

.background-btn {
  padding: 6px 12px;
  font-size: 12px;
  color: ${ne.textSecondary};
  background: ${ne.backgroundSecondary};
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  cursor: pointer;
  transition: all 0.2s ease;
}

.background-btn:hover {
  color: ${ne.primary};
  border-color: ${ne.primary};
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

.background-hint {
  font-size: 11px;
  color: ${ne.textMuted};
}

.canvas-container {
  position: relative;
  background: ${ne.backgroundSecondary};
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  overflow: hidden;
  box-shadow: ${ne.shadow};
  display: inline-block;
  max-width: min(100%, 1200px);
  max-height: min(80vh, 750px);
}

.canvas-container canvas {
  display: block;
  max-width: 100%;
  max-height: min(80vh, 750px);
  width: auto;
  height: auto;
}

/* 参数面板 */
.parameter-panel {
  width: 320px;
  background: ${ne.panelBackground};
  backdrop-filter: blur(${ne.glassBlur});
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  padding: 20px;
  overflow-y: auto;
  max-height: calc(100vh - 180px);
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: ${ne.primary};
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid ${ne.panelBorder};
  text-shadow: 0 0 8px ${ne.primaryGlow};
}

.parameter-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 参数控件通用样式 */
.param-control {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.param-label {
  font-size: 13px;
  color: ${ne.textSecondary};
  font-weight: 500;
}

/* 滑块控件 */
.slider-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.slider-input {
  flex: 1;
  -webkit-appearance: none;
  appearance: none;
  height: 6px;
  background: ${ne.backgroundSecondary};
  border-radius: 3px;
  outline: none;
}

.slider-input::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: ${ne.primary};
  cursor: pointer;
  box-shadow: 0 0 8px ${ne.primaryGlow};
  transition: all 0.2s ease;
}

.slider-input::-webkit-slider-thumb:hover {
  transform: scale(1.2);
  box-shadow: 0 0 12px ${ne.primaryGlow};
}

.slider-input::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border: none;
  border-radius: 50%;
  background: ${ne.primary};
  cursor: pointer;
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

.slider-value {
  font-size: 12px;
  color: ${ne.primary};
  min-width: 50px;
  text-align: right;
  font-family: 'SF Mono', Monaco, monospace;
}

/* 颜色控件 */
.color-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.color-input {
  width: 48px;
  height: 32px;
  border: 1px solid ${ne.panelBorder};
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  padding: 2px;
}

.color-input::-webkit-color-swatch-wrapper {
  padding: 0;
}

.color-input::-webkit-color-swatch {
  border: none;
  border-radius: 2px;
}

.color-value {
  font-size: 12px;
  color: ${ne.textSecondary};
  font-family: 'SF Mono', Monaco, monospace;
}

/* 开关控件 */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
}

.toggle-input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: ${ne.backgroundSecondary};
  border: 1px solid ${ne.panelBorder};
  border-radius: 26px;
  transition: all 0.3s ease;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 2px;
  bottom: 2px;
  background: ${ne.textMuted};
  border-radius: 50%;
  transition: all 0.3s ease;
}

.toggle-input:checked + .toggle-slider {
  background: rgba(0, 212, 255, 0.2);
  border-color: ${ne.primary};
}

.toggle-input:checked + .toggle-slider:before {
  transform: translateX(22px);
  background: ${ne.primary};
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

/* 下拉选择控件 */
.select-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 13px;
  color: ${ne.textPrimary};
  background: ${ne.backgroundSecondary};
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  outline: none;
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%2300d4ff' d='M2 4l4 4 4-4z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

.select-input:hover {
  border-color: ${ne.primary};
}

.select-input:focus {
  border-color: ${ne.primary};
  box-shadow: 0 0 0 2px ${ne.primaryGlow};
}

.select-input option {
  background: ${ne.background};
  color: ${ne.textPrimary};
}

/* 文本输入控件 (028-string-param) */
.param-text {
  margin-bottom: 4px;
}

.text-container {
  display: flex;
  align-items: center;
}

.text-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 13px;
  color: ${ne.textPrimary};
  background: ${ne.backgroundSecondary};
  border: 1px solid ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  outline: none;
  transition: all 0.2s ease;
}

.text-input:hover {
  border-color: ${ne.primary};
}

.text-input:focus {
  border-color: ${ne.primary};
  box-shadow: 0 0 0 2px ${ne.primaryGlow};
}

.text-input::placeholder {
  color: ${ne.textMuted};
}

/* 图片上传控件 */
.param-image {
  margin-bottom: 8px;
}

.image-upload-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-preview-wrapper {
  display: block;
  position: relative;
  width: 100%;
  height: 120px;
  background: ${ne.backgroundSecondary};
  border: 1px dashed ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
}

.image-preview-wrapper:hover {
  border-color: ${ne.primary};
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

.image-preview {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.image-placeholder {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: ${ne.textMuted};
}

.placeholder-icon {
  font-size: 32px;
  opacity: 0.5;
}

.placeholder-text {
  font-size: 12px;
}

.image-input {
  display: none;
}

.image-filename {
  font-size: 11px;
  color: ${ne.textMuted};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 视频上传控件 (019-video-input-support) */
.param-video {
  margin-bottom: 8px;
}

.video-upload-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.video-preview-wrapper {
  display: block;
  position: relative;
  width: 100%;
  height: 120px;
  background: ${ne.backgroundSecondary};
  border: 1px dashed ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
}

.video-preview-wrapper:hover {
  border-color: ${ne.primary};
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

.video-preview {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}

.video-placeholder {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: ${ne.textMuted};
  background: ${ne.backgroundSecondary};
}

.video-input {
  display: none;
}

.video-filename {
  font-size: 11px;
  color: ${ne.textMuted};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video-duration {
  font-size: 11px;
  color: ${ne.primary};
  font-family: 'SF Mono', Monaco, monospace;
}

/* T040: 序列帧上传控件 (029-sequence-frame-input) */
.param-sequence {
  margin-bottom: 8px;
}

.sequence-upload-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sequence-preview-wrapper {
  display: block;
  position: relative;
  width: 100%;
  height: 120px;
  background: ${ne.backgroundSecondary};
  border: 1px dashed ${ne.panelBorder};
  border-radius: ${ne.borderRadius};
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
}

.sequence-preview-wrapper:hover {
  border-color: ${ne.primary};
  box-shadow: 0 0 8px ${ne.primaryGlow};
}

.sequence-preview {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}

.sequence-placeholder {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: ${ne.textMuted};
  background: ${ne.backgroundSecondary};
}

.sequence-input {
  display: none;
}

.sequence-info {
  font-size: 11px;
  color: ${ne.textMuted};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 页脚 */
.app-footer {
  padding: 16px 24px;
  background: ${ne.panelBackground};
  backdrop-filter: blur(${ne.glassBlur});
  border-top: 1px solid ${ne.panelBorder};
  text-align: center;
}

.footer-text {
  font-size: 12px;
  color: ${ne.textMuted};
}

.footer-text strong {
  color: ${ne.primary};
}

/* 响应式布局 */
@media (max-width: 768px) {
  .main-content {
    flex-direction: column;
    padding: 16px;
  }

  .parameter-panel {
    width: 100%;
    max-height: 300px;
  }

  .canvas-container {
    width: 100%;
  }

  .canvas-container canvas {
    max-width: 100%;
    max-height: 40vh;
  }
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: ${ne.background};
}

::-webkit-scrollbar-thumb {
  background: ${ne.panelBorder};
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: ${ne.primary};
}

/* 无参数时的提示样式 */
.no-params-notice {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 40px 20px;
  text-align: center;
}

.no-params-notice .notice-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.no-params-notice .notice-text {
  color: ${ne.textMuted};
  font-size: 14px;
}

/* 浏览器兼容性警告 */
.compat-warning {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  padding: 12px 20px;
  background: rgba(255, 152, 0, 0.9);
  color: #000;
  text-align: center;
  font-size: 14px;
  z-index: 1000;
  display: none;
}

.compat-warning.show {
  display: block;
}
`.trim()}function ES(r){return r.length===0?"// 无参数控件需要绑定":`
// ============================================
// 参数控件事件绑定
// ============================================

(function() {
  'use strict';

${r.map(n=>CS(n)).join(`

`)}

})();
`.trim()}function CS(r){switch(r.controlType){case"slider":return bS(r);case"color":return SS(r);case"toggle":return RS(r);case"select":return AS(r);case"image":return PS(r);case"video":return kS(r);case"text":return DS(r);default:return""}}function bS(r){var n;const t=((n=r.numberConfig)==null?void 0:n.unit)||"";return`
  // ${r.label} - 滑块控件
  (function() {
    var slider = document.getElementById('param-${r.id}');
    var valueDisplay = document.getElementById('value-${r.id}');

    if (slider) {
      slider.addEventListener('input', function(e) {
        var value = parseFloat(e.target.value);
        if (valueDisplay) {
          valueDisplay.textContent = value + '${t}';
        }
        if (window.updateParam) {
          window.updateParam('${r.id}', value);
        }
      });
    }
  })();
`}function SS(r){return`
  // ${r.label} - 颜色控件
  (function() {
    var colorInput = document.getElementById('param-${r.id}');
    var valueDisplay = document.getElementById('value-${r.id}');

    if (colorInput) {
      colorInput.addEventListener('input', function(e) {
        var value = e.target.value;
        if (valueDisplay) {
          valueDisplay.textContent = value;
        }
        if (window.updateParam) {
          window.updateParam('${r.id}', value);
        }
      });
    }
  })();
`}function RS(r){return`
  // ${r.label} - 开关控件
  (function() {
    var toggle = document.getElementById('param-${r.id}');

    if (toggle) {
      toggle.addEventListener('change', function(e) {
        var value = e.target.checked;
        if (window.updateParam) {
          window.updateParam('${r.id}', value);
        }
      });
    }
  })();
`}function AS(r){return`
  // ${r.label} - 下拉选择控件
  (function() {
    var select = document.getElementById('param-${r.id}');

    if (select) {
      select.addEventListener('change', function(e) {
        var value = e.target.value;
        if (window.updateParam) {
          window.updateParam('${r.id}', value);
        }
      });
    }
  })();
`}function PS(r){return`
  // ${r.label} - 图片控件
  (function() {
    var fileInput = document.getElementById('param-${r.id}');
    var preview = document.getElementById('preview-${r.id}');
    var placeholder = document.getElementById('placeholder-${r.id}');
    var fileName = document.getElementById('filename-${r.id}');

    if (fileInput) {
      fileInput.addEventListener('change', function(e) {
        var file = e.target.files[0];
        if (!file) return;

        // 验证文件类型
        if (!file.type.match(/^image\\/(png|jpeg|jpg)$/)) {
          alert('仅支持 PNG 和 JPEG 格式的图片');
          return;
        }

        // 验证文件大小（最大 10MB）
        if (file.size > 10 * 1024 * 1024) {
          alert('图片文件过大，请选择小于 10MB 的图片');
          return;
        }

        // 显示文件名
        if (fileName) {
          fileName.textContent = file.name;
        }

        // 读取并加载图片
        var reader = new FileReader();
        reader.onload = function(event) {
          var img = new Image();
          img.onload = function() {
            // 更新预览，隐藏占位符
            if (preview) {
              preview.src = event.target.result;
              preview.style.display = 'block';
            }
            if (placeholder) {
              placeholder.style.display = 'none';
            }
            // 更新参数
            if (window.updateParam) {
              window.updateParam('${r.id}', img);
            }
          };
          img.onerror = function() {
            alert('图片加载失败，请选择其他图片');
          };
          img.src = event.target.result;
        };
        reader.readAsDataURL(file);
      });
    }
  })();
`}function kS(r){return`
  // ${r.label} - 视频控件
  (function() {
    var fileInput = document.getElementById('param-${r.id}');
    var preview = document.getElementById('preview-${r.id}');
    var placeholder = document.getElementById('placeholder-${r.id}');
    var fileName = document.getElementById('filename-${r.id}');
    var durationDisplay = document.getElementById('duration-${r.id}');

    if (fileInput) {
      fileInput.addEventListener('change', function(e) {
        var file = e.target.files[0];
        if (!file) return;

        // 验证文件类型
        if (!file.type.match(/^video\\/(mp4|webm)$/)) {
          alert('仅支持 MP4 和 WebM 格式的视频');
          return;
        }

        // 验证文件大小（最大 50MB）
        if (file.size > 50 * 1024 * 1024) {
          alert('视频文件过大，请选择小于 50MB 的视频');
          return;
        }

        // 显示文件名
        if (fileName) {
          fileName.textContent = file.name;
        }

        // 读取并加载视频
        var url = URL.createObjectURL(file);
        var video = document.createElement('video');
        video.muted = true;
        video.loop = true;
        video.playsInline = true;

        video.onloadedmetadata = function() {
          var duration = video.duration * 1000; // 转毫秒

          // 验证时长（最大 60 秒）
          if (duration > 60000) {
            URL.revokeObjectURL(url);
            alert('视频时长超过限制（最长 60 秒），请裁剪后重试');
            return;
          }

          // 开始播放视频（Canvas 渲染需要视频处于播放状态才能获取更新的帧）
          video.play().catch(function() {});

          // 更新预览，隐藏占位符
          if (preview) {
            preview.src = url;
            preview.style.display = 'block';
            preview.play().catch(function() {});
          }
          if (placeholder) {
            placeholder.style.display = 'none';
          }

          // 显示时长
          if (durationDisplay) {
            var secs = Math.round(duration / 1000);
            var mins = Math.floor(secs / 60);
            var secsRem = secs % 60;
            durationDisplay.textContent = mins > 0 ? (mins + ':' + (secsRem < 10 ? '0' : '') + secsRem) : (secsRem + '秒');
          }

          // 更新参数（传递正在播放的视频元素）
          // 注意：动态时长由 durationCode 控制，不再自动更新固定时长 (025-dynamic-duration)
          if (window.updateParam) {
            window.updateParam('${r.id}', video);
          }
        };

        video.onerror = function() {
          URL.revokeObjectURL(url);
          alert('视频加载失败，请检查文件是否损坏');
        };

        video.src = url;
        video.load();
      });
    }
  })();
`}function DS(r){return`
  // ${r.label} - 文本输入控件
  (function() {
    var textInput = document.getElementById('param-${r.id}');

    if (textInput) {
      textInput.addEventListener('input', function(e) {
        var value = e.target.value;
        if (window.updateParam) {
          window.updateParam('${r.id}', value);
        }
      });
    }
  })();
`}const TS={frameRate:60,quality:28,speed:5,keyframeInterval:1};class _S{generateEncoderService(t={}){const n={...TS,...t};return`
  // ============================================
  // H264 编码器服务 (与主平台 H264EncoderService 保持一致)
  // 使用 window. 暴露全局访问，供 RGB+Alpha 导出功能使用
  // ============================================

  window.H264EncoderService = {
    // 编码器实例
    encoder: null,
    isInitialized: false,
    isLoading: false,
    alignedWidth: 0,
    alignedHeight: 0,

    // 默认参数（与主平台一致）
    DEFAULT_FRAME_RATE: ${n.frameRate},
    DEFAULT_QUALITY: ${n.quality},
    DEFAULT_SPEED: ${n.speed},
    DEFAULT_KEYFRAME_INTERVAL: ${n.keyframeInterval},

    /**
     * 初始化编码器
     * @param width - 视频宽度
     * @param height - 视频高度
     * @param options - 编码选项
     */
    initialize: async function(width, height, options) {
      options = options || {};

      if (this.isInitialized) {
        console.warn('H264EncoderService: 编码器已初始化，请先调用 dispose()');
        return;
      }

      if (this.isLoading) {
        console.warn('H264EncoderService: 编码器正在加载中');
        return;
      }

      this.isLoading = true;

      try {
        // 检查 WASM 编码器是否可用
        if (typeof window.__initH264MP4Encoder__ !== 'function') {
          throw new Error('H264 MP4 编码器未正确加载');
        }

        // 创建编码器实例
        this.encoder = await window.__initH264MP4Encoder__();

        // 宽高必须是 2 的倍数（H.264 编码要求）
        this.alignedWidth = width % 2 === 0 ? width : width + 1;
        this.alignedHeight = height % 2 === 0 ? height : height + 1;

        // 设置编码器参数
        this.encoder.width = this.alignedWidth;
        this.encoder.height = this.alignedHeight;
        this.encoder.frameRate = options.frameRate || this.DEFAULT_FRAME_RATE;
        this.encoder.quantizationParameter = options.quality || this.DEFAULT_QUALITY;
        this.encoder.speed = options.speed || this.DEFAULT_SPEED;
        this.encoder.groupOfPictures = options.keyframeInterval || this.DEFAULT_KEYFRAME_INTERVAL;

        // 初始化编码器
        this.encoder.initialize();

        this.isInitialized = true;
        console.log('H264EncoderService: 编码器初始化成功', {
          width: this.alignedWidth,
          height: this.alignedHeight,
          frameRate: this.encoder.frameRate
        });

        return true;
      } catch (error) {
        console.error('H264EncoderService: 编码器初始化失败', error);
        throw error;
      } finally {
        this.isLoading = false;
      }
    },

    /**
     * 添加一帧
     * @param imageData - ImageData 对象
     */
    addFrame: function(imageData) {
      if (!this.isInitialized || !this.encoder) {
        throw new Error('编码器未初始化');
      }

      this.encoder.addFrameRgba(imageData.data);
    },

    /**
     * 添加一帧（从 Uint8Array）
     * @param rgbaData - RGBA 格式的原始数据
     */
    addFrameRgba: function(rgbaData) {
      if (!this.isInitialized || !this.encoder) {
        throw new Error('编码器未初始化');
      }

      this.encoder.addFrameRgba(rgbaData);
    },

    /**
     * 完成编码并返回 Blob
     */
    finalize: function() {
      if (!this.isInitialized || !this.encoder) {
        throw new Error('编码器未初始化');
      }

      // 完成编码
      this.encoder.finalize();

      // 读取输出文件
      var mp4Data = this.encoder.FS.readFile(this.encoder.outputFilename);

      // 创建 Blob
      var blob = new Blob([mp4Data], { type: 'video/mp4' });

      console.log('H264EncoderService: 编码完成', {
        size: blob.size,
        sizeKB: (blob.size / 1024).toFixed(1)
      });

      return blob;
    },

    /**
     * 释放编码器资源
     */
    dispose: function() {
      if (this.encoder) {
        try {
          this.encoder.delete();
          console.log('H264EncoderService: 编码器资源已释放');
        } catch (error) {
          console.warn('H264EncoderService: 释放编码器资源时出错', error);
        }
        this.encoder = null;
      }

      this.isInitialized = false;
      this.alignedWidth = 0;
      this.alignedHeight = 0;
    },

    /**
     * 获取对齐后的宽度
     */
    getAlignedWidth: function() {
      return this.alignedWidth;
    },

    /**
     * 获取对齐后的高度
     */
    getAlignedHeight: function() {
      return this.alignedHeight;
    },

    /**
     * 将值对齐到 2 的倍数
     */
    alignDimension: function(value) {
      return value % 2 === 0 ? value : value + 1;
    }
  };
`.trim()}generateVideoCloneService(){return`
  // ============================================
  // 视频克隆服务 (与主平台 VideoSyncService 保持一致)
  // 使用 window. 暴露全局访问，供 RGB+Alpha 导出功能使用
  // ============================================

  window.VideoCloneService = {
    /**
     * 克隆视频元素
     * @param original - 原始视频元素
     * @param timeout - 超时时间（毫秒），默认 5000
     */
    cloneVideo: function(original, timeout) {
      timeout = timeout || 5000;

      return new Promise(function(resolve, reject) {
        var clone = document.createElement('video');
        clone.src = original.src;
        clone.muted = true;
        clone.preload = 'auto';

        var cleanup = function() {
          clone.removeEventListener('loadeddata', onLoaded);
          clone.removeEventListener('error', onError);
        };

        var onLoaded = function() {
          cleanup();
          clearTimeout(timeoutId);
          console.log('VideoCloneService: 视频克隆成功');
          resolve(clone);
        };

        var onError = function() {
          cleanup();
          clearTimeout(timeoutId);
          console.error('VideoCloneService: 视频克隆失败');
          reject(new Error('视频克隆失败'));
        };

        clone.addEventListener('loadeddata', onLoaded, { once: true });
        clone.addEventListener('error', onError, { once: true });

        var timeoutId = setTimeout(function() {
          cleanup();
          console.warn('VideoCloneService: 视频克隆超时');
          reject(new Error('视频克隆超时'));
        }, timeout);
      });
    },

    /**
     * 批量克隆所有视频参数
     * @param params - 参数对象
     */
    cloneAllVideos: async function(params) {
      var result = {};

      for (var paramId in params) {
        var value = params[paramId];
        if (value && value.tagName === 'VIDEO') {
          try {
            result[paramId] = await this.cloneVideo(value);
          } catch (error) {
            console.warn('VideoCloneService: 跳过克隆失败的视频:', paramId);
            // 克隆失败时使用原始视频
            result[paramId] = value;
          }
        }
      }

      return result;
    },

    /**
     * 释放克隆的视频资源
     * @param clonedVideos - 克隆的视频映射
     */
    disposeClonedVideos: function(clonedVideos) {
      for (var paramId in clonedVideos) {
        var video = clonedVideos[paramId];
        if (video) {
          video.pause();
          video.src = '';
          video.load();
        }
      }
      console.log('VideoCloneService: 已释放克隆视频资源');
    }
  };
`.trim()}generateMP4ExporterCode(t={}){const n=this.generateEncoderService(t),a=this.generateVideoCloneService();return`
// ============================================
// MP4 视频导出器 (统一渲染架构 023-unified-renderer)
// ============================================

(function() {
  'use strict';

  var exportStatus = document.getElementById('export-status');

  // ==================== 状态显示 ====================

  function showStatus(text, isError) {
    if (exportStatus) {
      exportStatus.textContent = text;
      exportStatus.style.color = isError ? '#ff6b6b' : '#00d4ff';
      exportStatus.style.display = 'inline';
    }
  }

  function hideStatus() {
    if (exportStatus) {
      exportStatus.style.display = 'none';
    }
  }

${n}

${a}

  // ==================== 导出视频 ====================

  async function exportMP4Video() {
    if (!window.motionControls) {
      showStatus('动效控制器未初始化', true);
      return;
    }

    var canvas = document.getElementById('motion-canvas');
    if (!canvas) {
      showStatus('找不到画布元素', true);
      return;
    }

    // 禁用导出按钮
    var exportBtn = document.getElementById('export-btn');
    var originalHTML = '';
    if (exportBtn) {
      exportBtn.disabled = true;
      originalHTML = exportBtn.innerHTML;
      exportBtn.innerHTML = '<svg class="icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v4m0 12v4m-8-10H2m20 0h-2m-2.93-5.66l-1.41 1.41m-9.32 9.32l-1.41 1.41m0-12.02l1.41 1.41m9.32 9.32l1.41 1.41"/></svg>';
    }

    // 克隆的视频对象
    var clonedVideos = {};

    // 开始导出模式（暂停预览并保存状态）
    if (window.motionControls.beginExport) {
      window.motionControls.beginExport();
    } else if (window.motionControls.pause) {
      // 向后兼容：旧版本渲染器只有 pause 方法
      window.motionControls.pause();
    }
    // 更新播放按钮 UI
    var playBtn = document.getElementById('play-btn');
    if (playBtn) {
      var iconPlay = playBtn.querySelector('.icon-play');
      var iconPause = playBtn.querySelector('.icon-pause');
      if (iconPlay) iconPlay.style.display = 'block';
      if (iconPause) iconPause.style.display = 'none';
    }

    try {
      var width = canvas.width;
      var height = canvas.height;

      // 初始化编码器
      showStatus('正在初始化 MP4 编码器...', false);
      await H264EncoderService.initialize(width, height);

      var duration = window.motionControls.getDuration ? window.motionControls.getDuration() : 3000;
      var fps = H264EncoderService.DEFAULT_FRAME_RATE;
      var frameDelay = Math.round(1000 / fps);
      var totalFrames = Math.max(1, Math.round(duration / frameDelay));

      // 克隆所有视频参数
      var params = window.__motionParams;
      if (params) {
        showStatus('正在准备视频...', false);
        clonedVideos = await VideoCloneService.cloneAllVideos(params);
      }

      // 创建离屏 canvas
      var offCanvas = document.createElement('canvas');
      offCanvas.width = H264EncoderService.getAlignedWidth();
      offCanvas.height = H264EncoderService.getAlignedHeight();
      var offCtx = offCanvas.getContext('2d');

      // 逐帧捕获和编码
      for (var i = 0; i < totalFrames; i++) {
        var progress = totalFrames > 1 ? i / (totalFrames - 1) : 0;
        showStatus('捕获帧 ' + (i + 1) + '/' + totalFrames, false);

        var time = progress * duration;

        // 使用离屏 canvas 渲染帧
        if (typeof window.motionControls.renderToCanvas === 'function') {
          await window.motionControls.renderToCanvas(offCanvas, time, false, clonedVideos);
        } else if (typeof window.motionControls.renderAt === 'function') {
          await window.motionControls.renderAt(time, false);
          offCtx.drawImage(canvas, 0, 0, offCanvas.width, offCanvas.height);
        }

        // 获取 RGBA 数据并添加到编码器
        var imageData = offCtx.getImageData(0, 0, offCanvas.width, offCanvas.height);
        H264EncoderService.addFrame(imageData);

        // 每 10 帧让出主线程
        if (i % 10 === 0) {
          await new Promise(function(r) { setTimeout(r, 0); });
        }
      }

      showStatus('正在编码 MP4 视频...', false);

      // 完成编码
      var mp4Blob = H264EncoderService.finalize();

      // 释放编码器资源
      H264EncoderService.dispose();

      // 下载
      var url = URL.createObjectURL(mp4Blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = 'motion-animation.mp4';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      showStatus('导出成功！大小: ' + (mp4Blob.size / 1024).toFixed(1) + ' KB', false);
      setTimeout(hideStatus, 3000);

    } catch (err) {
      showStatus('导出失败: ' + err.message, true);
      setTimeout(hideStatus, 5000);

      // 清理编码器
      H264EncoderService.dispose();
    } finally {
      // 清理克隆的视频对象
      VideoCloneService.disposeClonedVideos(clonedVideos);

      // 结束导出模式（恢复预览到导出前状态）
      if (window.motionControls.endExport) {
        window.motionControls.endExport();
      }

      if (exportBtn) {
        exportBtn.disabled = false;
        exportBtn.innerHTML = originalHTML;
      }
    }
  }

  // 导出全局函数供统一导出调度器调用
  window.__exportMP4Video = exportMP4Video;
})();
`.trim()}}const LS="/neon/h264mp4/h264-mp4-encoder.web.js";let Ds=null;async function FS(){if(Ds)return Ds;const r=await fetch(LS);if(!r.ok)throw new Error(`Failed to load h264-mp4-encoder.web.js: ${r.status}`);return Ds=await r.text(),Ds}async function NS(){const r=await FS();if(!r)throw new Error("h264-mp4-encoder.web.js 加载内容为空");return["// ============================================","// H264 MP4 编码器（内嵌版本）","// ============================================","",r.replace(/^var HME=/,"window.HME="),"","// H264 编码器初始化辅助函数","window.__initH264MP4Encoder__ = function() {","  return window.HME.createH264MP4Encoder();","};"].join(`
`)}function IS(){return`
// ============================================
// WebGL 后处理运行时
// ============================================

(function() {
  var DEFAULT_VERTEX_SHADER = \`
    attribute vec2 aPosition;
    attribute vec2 aTexCoord;
    varying vec2 vUv;
    void main() {
      vUv = aTexCoord;
      gl_Position = vec4(aPosition, 0.0, 1.0);
    }
  \`;

  var PASSTHROUGH_FRAGMENT_SHADER = \`
    precision highp float;
    uniform sampler2D uTexture;
    varying vec2 vUv;
    void main() {
      gl_FragColor = texture2D(uTexture, vUv);
    }
  \`;

  function ensureShaderDeclarations(shader) {
    var uniforms = [];
    var varyings = [];
    if (shader.indexOf('uniform sampler2D uTexture') === -1) uniforms.push('uniform sampler2D uTexture;');
    if (shader.indexOf('uniform sampler2D uOriginal') === -1) uniforms.push('uniform sampler2D uOriginal;');
    if (shader.indexOf('uniform vec2 uResolution') === -1) uniforms.push('uniform vec2 uResolution;');
    if (shader.indexOf('uniform float uTime') === -1) uniforms.push('uniform float uTime;');
    if (shader.indexOf('varying vec2 vUv') === -1) varyings.push('varying vec2 vUv;');
    if (uniforms.length === 0 && varyings.length === 0) {
      if (shader.indexOf('precision ') === -1) return 'precision highp float;\\n\\n' + shader;
      return shader;
    }
    var processed = shader.replace(/precision\\s+(lowp|mediump|highp)\\s+float\\s*;/g, '');
    var decls = ['precision highp float;'].concat(uniforms).concat(varyings);
    return decls.join('\\n') + '\\n\\n' + processed;
  }

  function PostProcessRuntime() {
    this.gl = null;
    this.canvas = null;
    this.sourceTexture = null;
    this.pingPongTextures = [null, null];
    this.pingPongFramebuffers = [null, null];
    this.vertexBuffer = null;
    this.texCoordBuffer = null;
    this.programCache = {};
    this.passthroughProgram = null;
    this.width = 0;
    this.height = 0;
    this.disabled = false;
  }

  PostProcessRuntime.prototype.initialize = function(container, width, height) {
    this.width = width;
    this.height = height;
    this.canvas = document.createElement('canvas');
    this.canvas.width = width;
    this.canvas.height = height;
    container.appendChild(this.canvas);

    var gl = this.canvas.getContext('webgl', { alpha: true, premultipliedAlpha: false, preserveDrawingBuffer: true });
    if (!gl) {
      console.warn('[PostProcess] WebGL 不可用');
      this.disabled = true;
      return false;
    }
    this.gl = gl;
    this.initBuffers();
    this.initTextures();
    this.passthroughProgram = this.compileProgram('passthrough', PASSTHROUGH_FRAGMENT_SHADER);
    return true;
  };

  PostProcessRuntime.prototype.initBuffers = function() {
    var gl = this.gl;
    var vertices = new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]);
    this.vertexBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.vertexBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW);

    var texCoords = new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]);
    this.texCoordBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.texCoordBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, texCoords, gl.STATIC_DRAW);
  };

  PostProcessRuntime.prototype.initTextures = function() {
    var gl = this.gl;
    this.sourceTexture = this.createTexture();
    for (var i = 0; i < 2; i++) {
      var texture = this.createTexture();
      var fb = gl.createFramebuffer();
      gl.bindFramebuffer(gl.FRAMEBUFFER, fb);
      gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0);
      this.pingPongTextures[i] = texture;
      this.pingPongFramebuffers[i] = fb;
    }
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
  };

  PostProcessRuntime.prototype.createTexture = function() {
    var gl = this.gl;
    var texture = gl.createTexture();
    gl.bindTexture(gl.TEXTURE_2D, texture);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, this.width, this.height, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    return texture;
  };

  PostProcessRuntime.prototype.compileProgram = function(name, fragmentSrc) {
    var gl = this.gl;
    fragmentSrc = ensureShaderDeclarations(fragmentSrc);

    var vs = gl.createShader(gl.VERTEX_SHADER);
    gl.shaderSource(vs, DEFAULT_VERTEX_SHADER);
    gl.compileShader(vs);
    if (!gl.getShaderParameter(vs, gl.COMPILE_STATUS)) {
      console.error('Vertex shader error:', gl.getShaderInfoLog(vs));
      return null;
    }

    var fs = gl.createShader(gl.FRAGMENT_SHADER);
    gl.shaderSource(fs, fragmentSrc);
    gl.compileShader(fs);
    if (!gl.getShaderParameter(fs, gl.COMPILE_STATUS)) {
      console.error('Fragment shader error (' + name + '):', gl.getShaderInfoLog(fs));
      return null;
    }

    var program = gl.createProgram();
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      console.error('Program link error:', gl.getProgramInfoLog(program));
      return null;
    }

    gl.deleteShader(vs);
    gl.deleteShader(fs);
    return { program: program };
  };

  PostProcessRuntime.prototype.render = function(sourceCanvas, time, params, postProcessFn) {
    if (this.disabled || !this.gl) {
      var ctx = this.canvas.getContext('2d');
      if (ctx) ctx.drawImage(sourceCanvas, 0, 0);
      return;
    }

    var gl = this.gl;
    gl.bindTexture(gl.TEXTURE_2D, this.sourceTexture);
    // 翻转 Y 轴：Canvas 2D 的 Y 向下，WebGL 的 Y 向上
    gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, sourceCanvas);

    var passes = [];
    if (postProcessFn) {
      try { passes = postProcessFn(params, time); } catch (e) { console.error('postProcess error:', e); }
    }

    if (passes.length === 0) {
      this.renderPassthrough();
      return;
    }

    var inputTexture = this.sourceTexture;
    var ppIndex = 0;

    for (var i = 0; i < passes.length; i++) {
      var pass = passes[i];
      var isLast = i === passes.length - 1;
      var cacheKey = pass.name + '_' + pass.shader.length;
      var prog = this.programCache[cacheKey];
      if (!prog) {
        prog = this.compileProgram(pass.name, pass.shader);
        if (prog) this.programCache[cacheKey] = prog;
      }
      if (!prog) continue;

      if (isLast) {
        gl.bindFramebuffer(gl.FRAMEBUFFER, null);
      } else {
        gl.bindFramebuffer(gl.FRAMEBUFFER, this.pingPongFramebuffers[ppIndex]);
      }
      gl.viewport(0, 0, this.width, this.height);
      gl.clearColor(0, 0, 0, 0);
      gl.clear(gl.COLOR_BUFFER_BIT);

      gl.useProgram(prog.program);
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, inputTexture);

      // 绑定原始纹理到 TEXTURE1，用于 uOriginal
      gl.activeTexture(gl.TEXTURE1);
      gl.bindTexture(gl.TEXTURE_2D, this.sourceTexture);

      gl.uniform1i(gl.getUniformLocation(prog.program, 'uTexture'), 0);
      gl.uniform1i(gl.getUniformLocation(prog.program, 'uOriginal'), 1);
      gl.uniform2f(gl.getUniformLocation(prog.program, 'uResolution'), this.width, this.height);
      gl.uniform1f(gl.getUniformLocation(prog.program, 'uTime'), time);

      if (pass.uniforms) {
        for (var uName in pass.uniforms) {
          var uVal = pass.uniforms[uName];
          var uLoc = gl.getUniformLocation(prog.program, uName);
          if (!uLoc) continue;
          if (Array.isArray(uVal)) {
            if (uVal.length === 2) gl.uniform2fv(uLoc, uVal);
            else if (uVal.length === 3) gl.uniform3fv(uLoc, uVal);
            else if (uVal.length === 4) gl.uniform4fv(uLoc, uVal);
          } else {
            gl.uniform1f(uLoc, uVal);
          }
        }
      }

      var aPos = gl.getAttribLocation(prog.program, 'aPosition');
      if (aPos >= 0) {
        gl.bindBuffer(gl.ARRAY_BUFFER, this.vertexBuffer);
        gl.enableVertexAttribArray(aPos);
        gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);
      }
      var aTex = gl.getAttribLocation(prog.program, 'aTexCoord');
      if (aTex >= 0) {
        gl.bindBuffer(gl.ARRAY_BUFFER, this.texCoordBuffer);
        gl.enableVertexAttribArray(aTex);
        gl.vertexAttribPointer(aTex, 2, gl.FLOAT, false, 0, 0);
      }

      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);

      if (!isLast) {
        inputTexture = this.pingPongTextures[ppIndex];
        ppIndex = 1 - ppIndex;
      }
    }
  };

  PostProcessRuntime.prototype.renderPassthrough = function() {
    var gl = this.gl;
    if (!this.passthroughProgram) return;
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
    gl.viewport(0, 0, this.width, this.height);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.useProgram(this.passthroughProgram.program);
    gl.activeTexture(gl.TEXTURE0);
    gl.bindTexture(gl.TEXTURE_2D, this.sourceTexture);
    gl.uniform1i(gl.getUniformLocation(this.passthroughProgram.program, 'uTexture'), 0);
    var aPos = gl.getAttribLocation(this.passthroughProgram.program, 'aPosition');
    if (aPos >= 0) {
      gl.bindBuffer(gl.ARRAY_BUFFER, this.vertexBuffer);
      gl.enableVertexAttribArray(aPos);
      gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);
    }
    var aTex = gl.getAttribLocation(this.passthroughProgram.program, 'aTexCoord');
    if (aTex >= 0) {
      gl.bindBuffer(gl.ARRAY_BUFFER, this.texCoordBuffer);
      gl.enableVertexAttribArray(aTex);
      gl.vertexAttribPointer(aTex, 2, gl.FLOAT, false, 0, 0);
    }
    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
  };

  PostProcessRuntime.prototype.getCanvas = function() {
    return this.canvas;
  };

  PostProcessRuntime.prototype.resize = function(width, height) {
    this.width = width;
    this.height = height;
    this.canvas.width = width;
    this.canvas.height = height;
    if (this.gl) this.initTextures();
  };

  window.PostProcessRuntime = PostProcessRuntime;
})();
`.trim()}function BS(r){return!!(r.postProcessCode&&r.postProcessCode.trim())}const ht={TITLE:"{{TITLE}}",HEADER_TITLE:"{{HEADER_TITLE}}",STYLES:"{{STYLES}}",PARAMETER_PANEL:"{{PARAMETER_PANEL}}",EXPORT_TIME:"{{EXPORT_TIME}}",UTILS_CODE:"{{UTILS_CODE}}",IMAGE_PRELOAD_CODE:"{{IMAGE_PRELOAD_CODE}}",VIDEO_PRELOAD_CODE:"{{VIDEO_PRELOAD_CODE}}",RENDERER_CODE:"{{RENDERER_CODE}}",CONTROL_BINDINGS:"{{CONTROL_BINDINGS}}",POSTPROCESS_RUNTIME:"{{POSTPROCESS_RUNTIME}}",H264_CODE:"{{H264_CODE}}",MP4_EXPORTER:"{{MP4_EXPORTER}}"};async function jS(r,t,n,a){a==null||a(5);const i=await ox(n);a==null||a(15);const s=n.map(L=>L.parameter).filter(L=>L.type==="video"),c=await ax(s),u=Object.fromEntries(c.assets.map(L=>[L.paramId,L.base64Data]));a==null||a(25);const p=lS(n);a==null||a(30);let f;t.aspectRatio&&(f=Wl(t.aspectRatio,"1080p"));const h=gS(),x=hS(i),y=mS(u),w=pS(r,n,f);a==null||a(50);const E=wS();a==null||a(60);const C=OS(p,t.showPanelTitle);a==null||a(70);const S=ES(p);a==null||a(75);const P=BS(r)?IS():"// 无后处理";a==null||a(76);const b=new _S().generateMP4ExporterCode();a==null||a(80);const A=await NS();a==null||a(92);const T={exportedAt:Date.now()},B=qS(),F=t.customTitle||t.filename||"Motion Preview",k=KS(T.exportedAt),N=(L,M,H)=>L.replace(M,()=>H??"");let I=B;return I=N(I,ht.TITLE,ia(F)),I=N(I,ht.HEADER_TITLE,t.showPanelTitle?ia(F):""),I=N(I,ht.STYLES,E),I=N(I,ht.PARAMETER_PANEL,C),I=N(I,ht.EXPORT_TIME,k),I=N(I,ht.UTILS_CODE,h),I=N(I,ht.IMAGE_PRELOAD_CODE,x),I=N(I,ht.VIDEO_PRELOAD_CODE,y),I=N(I,ht.RENDERER_CODE,w),I=N(I,ht.CONTROL_BINDINGS,S),I=N(I,ht.POSTPROCESS_RUNTIME,P),I=N(I,ht.H264_CODE,A),I=N(I,ht.MP4_EXPORTER,b),a==null||a(100),I}function OS(r,t){if(r.length===0)return"<!-- 无可调参数 -->";const n=r.map(a=>MS(a)).join(`
`);return`
      <aside class="parameter-panel">
        ${t?'<h2 class="panel-title">参数调整</h2>':""}
        <div class="parameter-list">
          ${n}
        </div>
      </aside>
  `}function MS(r){const t=`<label class="param-label" for="param-${r.id}">${ia(r.label)}</label>`;switch(r.controlType){case"slider":return $S(r,t);case"color":return US(r,t);case"toggle":return zS(r,t);case"select":return HS(r,t);case"image":return VS(r,t);case"video":return WS(r,t);case"text":return GS(r,t);default:return""}}function $S(r,t){const{min:n=0,max:a=100,step:i=1,unit:s=""}=r.numberConfig||{},c=r.initialValue;return`
          <div class="param-control param-slider">
            ${t}
            <div class="slider-container">
              <input
                type="range"
                id="param-${r.id}"
                data-param-id="${r.id}"
                min="${n}"
                max="${a}"
                step="${i}"
                value="${c}"
                class="slider-input"
              />
              <span class="slider-value" id="value-${r.id}">${c}${s}</span>
            </div>
          </div>
  `}function US(r,t){const n=r.initialValue;return`
          <div class="param-control param-color">
            ${t}
            <div class="color-container">
              <input
                type="color"
                id="param-${r.id}"
                data-param-id="${r.id}"
                value="${n}"
                class="color-input"
              />
              <span class="color-value" id="value-${r.id}">${n}</span>
            </div>
          </div>
  `}function zS(r,t){const n=r.initialValue;return`
          <div class="param-control param-toggle">
            ${t}
            <label class="toggle-switch">
              <input
                type="checkbox"
                id="param-${r.id}"
                data-param-id="${r.id}"
                ${n?"checked":""}
                class="toggle-input"
              />
              <span class="toggle-slider"></span>
            </label>
          </div>
  `}function HS(r,t){var s;const n=r.initialValue,i=(((s=r.selectConfig)==null?void 0:s.options)||[]).map(c=>`<option value="${ia(c.value)}" ${c.value===n?"selected":""}>${ia(c.label)}</option>`).join(`
`);return`
          <div class="param-control param-select">
            ${t}
            <select
              id="param-${r.id}"
              data-param-id="${r.id}"
              class="select-input"
            >
              ${i}
            </select>
          </div>
  `}function VS(r,t){const n=r.initialValue,a=n&&n.startsWith("data:");return`
          <div class="param-control param-image">
            ${t}
            <div class="image-upload-container">
              <label class="image-preview-wrapper">
                <input
                  type="file"
                  id="param-${r.id}"
                  data-param-id="${r.id}"
                  accept="image/png,image/jpeg"
                  class="image-input"
                />
                <img
                  id="preview-${r.id}"
                  class="image-preview"
                  src="${a?n:""}"
                  style="display: ${a?"block":"none"};"
                  alt="预览"
                />
                <div class="image-placeholder" id="placeholder-${r.id}" style="display: ${a?"none":"flex"};">
                  <span class="placeholder-icon">🖼️</span>
                  <span class="placeholder-text">点击上传图片</span>
                </div>
              </label>
              <span class="image-filename" id="filename-${r.id}"></span>
            </div>
          </div>
  `}function WS(r,t){const n=r.initialValue,a=n&&n.startsWith("data:");return`
          <div class="param-control param-video">
            ${t}
            <div class="video-upload-container">
              <label class="video-preview-wrapper">
                <input
                  type="file"
                  id="param-${r.id}"
                  data-param-id="${r.id}"
                  accept="video/mp4,video/webm"
                  class="video-input"
                />
                <video
                  id="preview-${r.id}"
                  class="video-preview"
                  muted
                  loop
                  playsinline
                  src="${a?n:""}"
                  style="display: ${a?"block":"none"};"
                ></video>
                <div class="video-placeholder" id="placeholder-${r.id}" style="display: ${a?"none":"flex"};">
                  <span class="placeholder-icon">🎬</span>
                  <span class="placeholder-text">点击上传视频</span>
                </div>
              </label>
              <span class="video-filename" id="filename-${r.id}"></span>
              <span class="video-duration" id="duration-${r.id}"></span>
            </div>
          </div>
  `}function GS(r,t){const n=r.initialValue;return`
          <div class="param-control param-text">
            ${t}
            <div class="text-container">
              <input
                type="text"
                id="param-${r.id}"
                data-param-id="${r.id}"
                value="${ia(n)}"
                class="text-input"
                placeholder=""
              />
            </div>
          </div>
  `}function qS(){return`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${ht.TITLE}</title>
  <style>
    ${ht.STYLES}
  </style>
</head>
<body>
  <div class="app-container">
    <!-- 头部区域 -->
    <header class="app-header">
      <h1 class="app-title">${ht.HEADER_TITLE}</h1>
      <div class="header-controls">
        <button id="play-btn" class="control-btn" title="播放/暂停">
          <svg class="icon icon-play" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
          <svg class="icon icon-pause" viewBox="0 0 24 24" fill="currentColor" style="display:none;">
            <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
          </svg>
        </button>
        <button id="stop-btn" class="control-btn" title="停止">
          <svg class="icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M6 6h12v12H6z"/>
          </svg>
        </button>
        <div class="header-divider"></div>
        <div class="export-group">
          <button id="export-btn" class="export-btn" title="导出动画">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            <span>导出 MP4</span>
          </button>
        </div>
        <span id="export-status" class="export-status"></span>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="main-content">
      <!-- Canvas 预览区 -->
      <div class="preview-area">
        <!-- T011-T013: 背景图容器 (033-preview-background) -->
        <div class="background-container" id="background-container" style="background-image: none; background-size: cover; background-position: center; background-repeat: no-repeat;">
          <div class="canvas-container" id="canvas-container">
            <canvas id="motion-canvas"></canvas>
          </div>
        </div>
        <!-- T011: 背景图上传按钮 (033-preview-background) -->
        <div class="background-controls">
          <input type="file" id="background-input" accept="image/png,image/jpeg,image/webp" style="display: none;" />
          <button id="upload-background-btn" class="background-btn" title="上传预览背景图">上传背景图</button>
          <button id="clear-background-btn" class="background-btn" style="display: none;" title="清除背景图">清除背景</button>
          <span class="background-hint">背景图仅预览使用，不参与导出</span>
        </div>
      </div>

      <!-- 参数面板 -->
      ${ht.PARAMETER_PANEL}
    </main>

    <!-- 页脚 -->
    <footer class="app-footer">
      <p class="footer-text">
        导出自 <strong>Neon Motion Platform</strong> · ${ht.EXPORT_TIME}
      </p>
    </footer>
  </div>

  <!-- 工具函数 -->
  <script>
    ${ht.UTILS_CODE}
  <\/script>

  <!-- 后处理运行时 -->
  <script>
    ${ht.POSTPROCESS_RUNTIME}
  <\/script>

  <!-- 渲染器代码 -->
  <script>
    ${ht.RENDERER_CODE}
  <\/script>

  <!-- 视频资源预加载 -->
  <script>
    ${ht.VIDEO_PRELOAD_CODE}
  <\/script>

  <!-- 图片预加载 -->
  <script>
    ${ht.IMAGE_PRELOAD_CODE}
  <\/script>

  <!-- 控件绑定代码 -->
  <script>
    ${ht.CONTROL_BINDINGS}
  <\/script>

  <!-- H264 MP4 编码器（内嵌） -->
  <script>
    ${ht.H264_CODE}
  <\/script>

  <!-- MP4 导出功能 -->
  <script>
    ${ht.MP4_EXPORTER}
  <\/script>

  <!-- 导出按钮绑定 -->
  <script>
    (function() {
      var exportBtn = document.getElementById('export-btn');
      if (!exportBtn) return;
      exportBtn.addEventListener('click', function() {
        if (typeof window.__exportMP4Video === 'function') {
          window.__exportMP4Video();
        }
      });
    })();
  <\/script>

  <!-- 播放控制绑定 -->
  <script>
    (function() {
      var playBtn = document.getElementById('play-btn');
      var stopBtn = document.getElementById('stop-btn');
      var iconPlay = playBtn.querySelector('.icon-play');
      var iconPause = playBtn.querySelector('.icon-pause');

      function updatePlayIcon(isPlaying) {
        iconPlay.style.display = isPlaying ? 'none' : 'block';
        iconPause.style.display = isPlaying ? 'block' : 'none';
      }

      playBtn.addEventListener('click', function() {
        if (window.motionControls.isPlaying()) {
          window.motionControls.pause();
          updatePlayIcon(false);
        } else {
          window.motionControls.play();
          updatePlayIcon(true);
        }
      });

      stopBtn.addEventListener('click', function() {
        window.motionControls.stop();
        updatePlayIcon(false);
      });

      // 初始状态
      updatePlayIcon(true);
    })();
  <\/script>

  <!-- T012-T014: 背景图上传逻辑 (033-preview-background) -->
  <script>
    (function() {
      'use strict';

      var backgroundContainer = document.getElementById('background-container');
      var canvasContainer = document.getElementById('canvas-container');
      var backgroundInput = document.getElementById('background-input');
      var uploadBtn = document.getElementById('upload-background-btn');
      var clearBtn = document.getElementById('clear-background-btn');
      var currentBackgroundUrl = null;

      if (!backgroundContainer || !canvasContainer || !backgroundInput || !uploadBtn || !clearBtn) {
        console.warn('[Background] 背景图控件元素未找到');
        return;
      }

      // 保存 canvas-container 的原始背景
      var originalCanvasBg = window.getComputedStyle(canvasContainer).background;

      // T012: 上传按钮点击触发 file input
      uploadBtn.addEventListener('click', function() {
        backgroundInput.click();
      });

      // T012: 文件选择处理
      backgroundInput.addEventListener('change', function(e) {
        var file = e.target.files && e.target.files[0];
        if (!file) return;

        // 文件类型校验（PNG/JPG/WebP）
        var validTypes = ['image/png', 'image/jpeg', 'image/webp'];
        if (validTypes.indexOf(file.type) === -1) {
          alert('仅支持 PNG、JPG、WebP 格式的图片');
          backgroundInput.value = '';
          return;
        }

        // 大文件警告（>10MB）
        if (file.size > 10 * 1024 * 1024) {
          console.warn('[Background] 背景图较大，建议使用小于 10MB 的图片');
        }

        // 释放旧的 Blob URL
        if (currentBackgroundUrl) {
          URL.revokeObjectURL(currentBackgroundUrl);
        }

        // T013: 创建 Blob URL 并应用 CSS 背景图样式
        currentBackgroundUrl = URL.createObjectURL(file);
        backgroundContainer.style.backgroundImage = 'url("' + currentBackgroundUrl + '")';

        // 将 canvas-container 背景设为透明，让背景图可见
        canvasContainer.style.background = 'transparent';

        // 显示清除按钮
        clearBtn.style.display = 'inline-block';

        // 重置 input 以支持再次选择同一文件
        backgroundInput.value = '';

        console.log('[Background] 背景图已设置');
      });

      // T014: 清除背景图
      clearBtn.addEventListener('click', function() {
        // 释放 Blob URL
        if (currentBackgroundUrl) {
          URL.revokeObjectURL(currentBackgroundUrl);
          currentBackgroundUrl = null;
        }

        // 清除 CSS 背景图
        backgroundContainer.style.backgroundImage = 'none';

        // 恢复 canvas-container 原始背景
        canvasContainer.style.background = originalCanvasBg;

        // 隐藏清除按钮
        clearBtn.style.display = 'none';

        console.log('[Background] 背景图已清除');
      });
    })();
  <\/script>

  <!-- 浏览器兼容性检测 -->
  <script>
    (function() {
      var warnings = [];

      // 检查 Canvas 支持
      var testCanvas = document.createElement('canvas');
      if (!testCanvas.getContext || !testCanvas.getContext('2d')) {
        warnings.push('您的浏览器不支持 Canvas，动效可能无法正常显示');
      }

      // 检查 requestAnimationFrame 支持
      if (!window.requestAnimationFrame) {
        warnings.push('您的浏览器不支持流畅动画，建议升级浏览器');
      }

      // 显示警告
      if (warnings.length > 0) {
        var warningDiv = document.createElement('div');
        warningDiv.className = 'compat-warning show';
        warningDiv.textContent = warnings.join(' | ');
        document.body.insertBefore(warningDiv, document.body.firstChild);
      }
    })();
  <\/script>
</body>
</html>`}function ia(r){if(!r)return"";const t={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"};return r.replace(/[&<>"']/g,n=>t[n]||n)}function KS(r){return new Date(r).toLocaleString("zh-CN",{year:"numeric",month:"2-digit",day:"2-digit",hour:"2-digit",minute:"2-digit"})}class XS{prepareExport(t){return Ya(t)}async generateHtml(t,n,a){const i=Ya(t),s=rd(i,n.selectedParameterIds);return jS(t,n,s,a)}downloadHtml(t,n){const a=n.endsWith(".html")?n:`${n}.html`,i=new Blob([t],{type:"text/html;charset=utf-8"}),s=URL.createObjectURL(i),c=document.createElement("a");c.href=s,c.download=a,document.body.appendChild(c),c.click(),document.body.removeChild(c),URL.revokeObjectURL(s)}estimateFileSize(t,n){const a=Ya(t),i=rd(a,n.selectedParameterIds);let s=10*1024;s+=5*1024,s+=2*1024,s+=t.code.length;const c=i.filter(p=>p.selected).length;s+=c*500;const u=i.filter(p=>p.parameter.type==="image").length;return s+=u*67*1024,s}async estimateFileSizeAsync(t,n){const a=Ya(t),i=rd(a,n.selectedParameterIds);let s=10*1024+5*1024+2*1024+t.code.length;const c=i.filter(u=>u.selected).length;s+=c*500;try{const u=await ox(i);s+=vS(u)}catch{const u=i.filter(p=>p.parameter.type==="image").length;s+=u*67*1024}try{const u=i.map(f=>f.parameter),p=await ax(u);s+=p.totalSize;for(const f of p.assets)f.originalSize>50*1024*1024&&console.warn(`[AssetPackExporter] 视频 ${f.paramId} 超过 50MB，导出文件将较大`);p.totalSize>200*1024*1024&&console.warn("[AssetPackExporter] 总视频大小超过 200MB，导出可能较慢");for(const f of p.errors)console.warn(`[AssetPackExporter] 视频资源收集失败: ${f.paramId} - ${f.message}`)}catch{const u=i.filter(p=>p.parameter.type==="video").length;s+=u*50*1024*1024}return s}}const nd=new XS;function YS(r){return r<1024?`${r} B`:r<1024*1024?`${(r/1024).toFixed(1)} KB`:`${(r/(1024*1024)).toFixed(2)} MB`}const QS=()=>{const{t:r}=ft(),{isAssetPackExportDialogOpen:t,assetPackExportState:n,currentMotion:a,aspectRatio:i,closeAssetPackExportDialog:s,setAssetPackExportState:c}=Ht(),[u,p]=D.useState([]),[f,h]=D.useState("motion-preview"),[x,y]=D.useState(!0),[w,E]=D.useState(""),[C,S]=D.useState(0);D.useEffect(()=>{if(t&&a){const I=Ya(a);p(I),h(a.id||"motion-preview"),E("")}},[t,a]),D.useEffect(()=>{if(a&&u.length>0){const I={filename:f,selectedParameterIds:Dm(u),showPanelTitle:x,customTitle:w||void 0,aspectRatio:i};S(nd.estimateFileSize(a,I))}},[u,f,x,w,a,i]);const P=D.useCallback(I=>{p(L=>dS(L,I))},[]),b=D.useCallback(()=>{p(I=>cS(I))},[]),A=D.useCallback(()=>{p(I=>uS(I))},[]),T=D.useCallback(async()=>{if(!a)return;const I={filename:f||"motion-preview",selectedParameterIds:Dm(u),showPanelTitle:x,customTitle:w||void 0,aspectRatio:i};try{c({status:"generating",config:I,progress:0,error:null});const L=await nd.generateHtml(a,I,M=>{c({progress:M})});c({status:"downloading",progress:100}),nd.downloadHtml(L,I.filename),c({status:"completed"}),setTimeout(()=>{s()},500)}catch(L){console.error("Export failed:",L),c({status:"error",error:L instanceof Error?L.message:r("export.failed")})}},[r,a,f,u,x,w,i,c,s]);D.useEffect(()=>{const I=L=>{L.key==="Escape"&&t&&s()};return t&&document.addEventListener("keydown",I),()=>{document.removeEventListener("keydown",I)}},[t,s]);const B=I=>{I.target===I.currentTarget&&n.status!=="generating"&&s()};if(!t)return null;const F=n.status==="generating"||n.status==="downloading",k=n.status==="error",N=u.filter(I=>I.selected).length;return g.jsx("div",{className:"fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm",onClick:B,children:g.jsxs("div",{className:"bg-background-elevated rounded-lg shadow-neon-soft w-full max-w-lg mx-4 overflow-hidden border border-border-default",role:"dialog","aria-modal":"true","aria-labelledby":"export-dialog-title",children:[g.jsxs("div",{className:"px-6 py-4 border-b border-border-default",children:[g.jsx("h2",{id:"export-dialog-title",className:"text-lg font-semibold font-display text-text-primary",children:r("export.assetPackTitle")}),g.jsx("p",{className:"text-sm font-body text-text-muted mt-1",children:r("exportAssetPack.subtitle")})]}),g.jsxs("div",{className:"px-6 py-4 max-h-[60vh] overflow-y-auto",children:[g.jsxs("div",{className:"mb-5",children:[g.jsx("label",{className:"block text-sm font-medium font-body text-text-muted mb-2",children:r("exportAssetPack.filename")}),g.jsxs("div",{className:"flex items-center gap-2",children:[g.jsx("input",{type:"text",value:f,onChange:I=>h(I.target.value),placeholder:"motion-preview",disabled:F,className:"flex-1 px-3 py-2 bg-background-secondary border border-border-default rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-accent-primary focus:border-transparent disabled:opacity-50 transition-colors duration-200"}),g.jsx("span",{className:"text-sm font-body text-text-muted",children:".html"})]})]}),g.jsxs("div",{className:"mb-5",children:[g.jsx("label",{className:"block text-sm font-medium font-body text-text-muted mb-2",children:r("exportAssetPack.customTitle")}),g.jsx("input",{type:"text",value:w,onChange:I=>E(I.target.value),placeholder:r("exportAssetPack.customTitlePlaceholder"),disabled:F,className:"w-full px-3 py-2 bg-background-secondary border border-border-default rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-accent-primary focus:border-transparent disabled:opacity-50 transition-colors duration-200"})]}),g.jsx("div",{className:"mb-5",children:g.jsxs("label",{className:"flex items-center gap-3 cursor-pointer",children:[g.jsx("input",{type:"checkbox",checked:x,onChange:I=>y(I.target.checked),disabled:F,className:"w-4 h-4 accent-accent-primary cursor-pointer"}),g.jsx("span",{className:"text-sm font-body text-text-primary",children:r("exportAssetPack.showTitle")})]})}),g.jsxs("div",{className:"mb-4",children:[g.jsx("label",{className:"block text-sm font-medium font-body text-text-muted mb-3",children:r("exportAssetPack.selectParameters")}),g.jsx(aS,{parameters:u,onToggle:P,onSelectAll:b,onDeselectAll:A})]}),g.jsxs("div",{className:"text-xs font-body text-text-muted flex items-center gap-2",children:[g.jsx("span",{children:r("exportAssetPack.estimatedSize")}),g.jsx("span",{className:"font-mono text-accent-primary",children:YS(C)})]}),k&&n.error&&g.jsx("div",{className:"mt-4 p-3 bg-accent-tertiary/10 border border-accent-tertiary/30 rounded-lg text-accent-tertiary text-sm",children:n.error}),F&&g.jsxs("div",{className:"mt-4",children:[g.jsxs("div",{className:"flex items-center justify-between text-sm mb-2",children:[g.jsx("span",{className:"font-body text-text-muted",children:n.status==="generating"?r("exportAssetPack.generating"):r("exportAssetPack.downloading")}),g.jsxs("span",{className:"text-accent-primary",children:[n.progress,"%"]})]}),g.jsx("div",{className:"h-2 bg-background-secondary rounded-full overflow-hidden",children:g.jsx("div",{className:"h-full bg-accent-primary transition-all duration-300",style:{width:`${n.progress}%`}})})]})]}),g.jsxs("div",{className:"px-6 py-4 border-t border-border-default bg-background-secondary flex items-center justify-between",children:[g.jsx("div",{className:"text-sm font-body text-text-muted",children:N>0?g.jsx("span",{children:r("exportAssetPack.selectedCount",{count:N})}):g.jsx("span",{children:r("exportAssetPack.noParameters")})}),g.jsxs("div",{className:"flex gap-3",children:[g.jsx(lt,{variant:"ghost",size:"sm",onClick:s,disabled:F,children:r("common.cancel")}),g.jsx(lt,{variant:"primary",size:"sm",onClick:T,disabled:F||!a,loading:F,children:r(F?"exportAssetPack.exportingButton":"exportAssetPack.exportButton")})]})]})]})})};function JS(){const{currentMotion:r,isPlaying:t,setIsPlaying:n,openSettings:a,openExportDialog:i}=Ht(),s=D.useCallback(c=>{const u=c.target;if(!(u.tagName==="INPUT"||u.tagName==="TEXTAREA"||u.isContentEditable))switch(c.code){case"Space":r&&(c.preventDefault(),n(!t));break;case"KeyE":(c.metaKey||c.ctrlKey)&&r&&(c.preventDefault(),i());break;case"Comma":(c.metaKey||c.ctrlKey)&&(c.preventDefault(),a());break}},[r,t,n,a,i]);D.useEffect(()=>(window.addEventListener("keydown",s),()=>{window.removeEventListener("keydown",s)}),[s])}function ZS(){const{t:r}=ft(),{llmConfigs:t,activeConfigId:n,isLoadingConfigs:a,currentMotion:i,isSettingsOpen:s,isExportDialogOpen:c,locale:u,setLocale:p,openSettings:f,closeSettings:h,openExportDialog:x,closeExportDialog:y,openAssetPackExportDialog:w,loadFromStorage:E,initConversations:C,initLLMConfigs:S}=Ht(),P=t.length>0&&n!==null;return D.useEffect(()=>{E(),C(),S()},[E,C,S]),JS(),g.jsx(jl,{children:g.jsxs("div",{className:"h-screen flex flex-col bg-background-primary",children:[g.jsxs("header",{className:"relative h-14 bg-background-elevated border-b border-border-default flex items-center justify-between px-4 shrink-0",children:[g.jsxs("div",{className:"flex items-center gap-4",children:[g.jsx(Xn,{to:"/",className:"p-1 -ml-1 text-text-muted hover:text-accent-primary transition-colors rounded hover:bg-accent-primary/10",title:"返回首页",children:g.jsx("svg",{className:"w-5 h-5",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M10 19l-7-7m0 0l7-7m-7 7h18"})})}),g.jsx("h1",{className:"text-lg sm:text-xl font-display text-accent-primary",style:{textShadow:"var(--text-glow)"},children:"Neon Lab"})]}),g.jsxs("div",{className:"flex gap-2",children:[g.jsx("button",{onClick:()=>p(u==="zh"?"en":"zh"),className:"px-2 py-1 text-xs font-medium rounded-md border border-border-default text-text-muted hover:text-accent-primary hover:border-accent-primary transition-colors cursor-pointer",title:u==="zh"?"Switch to English":"切换为中文",children:u==="zh"?"EN":"中"}),g.jsxs(lt,{variant:"ghost",size:"sm",onClick:f,className:s?"bg-accent-primary/15 text-accent-primary":"",children:[g.jsx("span",{className:"hidden sm:inline",children:r("nav.settings")}),g.jsxs("svg",{className:"w-5 h-5 sm:hidden",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:[g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426-1.756-2.924-1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"}),g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M15 12a3 3 0 11-6 0 3 3 0 016 0z"})]})]}),g.jsxs(lt,{variant:"secondary",size:"sm",onClick:w,disabled:!i,title:"导出为可独立运行的 HTML 素材包",children:[g.jsx("span",{className:"hidden sm:inline",children:r("nav.exportAssetPack")}),g.jsx("svg",{className:"w-5 h-5 sm:hidden",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7l8 4"})})]}),g.jsxs(lt,{variant:"primary",size:"sm",onClick:x,disabled:!i,children:[g.jsx("span",{className:"hidden sm:inline",children:r("nav.exportVideo")}),g.jsx("svg",{className:"w-5 h-5 sm:hidden",fill:"none",stroke:"currentColor",viewBox:"0 0 24 24",children:g.jsx("path",{strokeLinecap:"round",strokeLinejoin:"round",strokeWidth:2,d:"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"})})]})]})]}),g.jsxs("main",{className:"flex-1 flex flex-col lg:flex-row overflow-hidden",children:[g.jsxs("div",{className:"w-full lg:w-80 bg-background-secondary border-b lg:border-b-0 lg:border-r border-border-default flex flex-col shrink-0",children:[g.jsx("div",{className:"p-3 border-b border-border-default shrink-0",children:g.jsx("h2",{className:"text-sm font-medium font-display text-text-primary",children:r("panel.conversations")})}),!P&&!a?g.jsx("div",{className:"flex-1 flex items-center justify-center",children:g.jsxs("div",{className:"text-center text-text-muted px-4",children:[g.jsx("p",{className:"mb-3 font-body",children:r("chat.configRequired")}),g.jsx(lt,{variant:"secondary",size:"sm",onClick:f,children:r("nav.openSettings")})]})}):g.jsx("div",{className:"flex-1 min-h-0",children:g.jsx(V4,{})})]}),g.jsx("div",{className:"flex-1 flex flex-col min-w-0 bg-background-elevated",children:g.jsx(Hb,{})}),g.jsxs("div",{className:"w-full lg:w-72 bg-background-secondary border-t lg:border-t-0 lg:border-l border-border-default flex flex-col shrink-0",children:[g.jsx("div",{className:"p-3 border-b border-border-default shrink-0",children:g.jsx("h2",{className:"text-sm font-medium font-display text-text-primary",children:r("panel.parameters")})}),g.jsxs("div",{className:"flex-1 overflow-y-auto p-3 min-h-0",children:[g.jsx("div",{className:"mb-4 pb-4 border-b border-border-default",children:g.jsx(W4,{})}),g.jsx(Jb,{})]})]})]}),g.jsx(X4,{isOpen:s,onClose:h}),g.jsx(nS,{isOpen:c,onClose:y}),g.jsx(QS,{})]})})}function eR(){return g.jsx(jl,{children:g.jsxs("div",{className:"min-h-screen bg-background-primary flex items-center justify-center",children:[g.jsx(Bd,{}),g.jsxs("main",{className:"flex-1 flex flex-col items-center justify-center px-4 -mt-16",children:[g.jsx("h1",{className:"font-display text-8xl md:text-9xl text-accent-primary mb-4",style:{textShadow:"var(--glow-medium)"},children:"404"}),g.jsx("h2",{className:"font-display text-2xl md:text-3xl text-text-primary mb-4",children:"页面未找到"}),g.jsx("p",{className:"font-body text-text-muted mb-8 text-center",children:"您访问的页面不存在或已被移除"}),g.jsx(Xn,{to:"/",className:"px-6 py-2.5 bg-accent-primary text-black rounded-lg font-medium cursor-pointer hover:bg-accent-primary/90 transition-all duration-200 inline-block",children:"返回首页"})]})]})})}const tR=Zw([{path:"/",element:g.jsx($C,{})},{path:"/neon-lab",element:g.jsx(ZS,{})},{path:"/demos",element:g.jsx(nb,{})},{path:"*",element:g.jsx(eR,{})}]);function rR(){return g.jsx(p2,{router:tR})}localStorage.removeItem("motion-platform:theme-preference");Zv.createRoot(document.getElementById("root")).render(g.jsx(D.StrictMode,{children:g.jsx(rR,{})}));
