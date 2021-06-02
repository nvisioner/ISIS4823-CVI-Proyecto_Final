#version 410 core
out vec4 FragColor;


struct PointLight {
    vec3 position;
    float constant;
    float linear;
    float quadratic;
	vec3 lightColor;
    vec3 ambient;
    vec3 diffuse;
    vec3 specular;
};



#define NR_POINT_LIGHTS 8

in vec3 FragPos;
in vec3 Normal;
in vec2 TexCoord;
in mat3 TBN;

uniform float time;
uniform int numLights;
uniform vec3 objectColor;
uniform vec3 viewPos;
uniform sampler2D texture_diffuse1;
uniform sampler2D texture_normal1;
uniform PointLight pointLights[NR_POINT_LIGHTS];

precision mediump int;
precision mediump float;

vec4 mod289(vec4 x)
{
  return x - floor(x * (1.0 / 289.0)) * 289.0;
}

vec4 permute(vec4 x)
{
  return mod289(((x*34.0)+1.0)*x);
}

vec4 taylorInvSqrt(vec4 r)
{
  return 1.79284291400159 - 0.85373472095314 * r;
}

vec2 fade(vec2 t) {
  return t*t*t*(t*(t*6.0-15.0)+10.0);
}

// Classic Perlin noise, periodic variant
float pnoise(vec2 P, vec2 rep)
{
  vec4 Pi = floor(P.xyxy) + vec4(0.0, 0.0, 1.0, 1.0);
  vec4 Pf = fract(P.xyxy) - vec4(0.0, 0.0, 1.0, 1.0);
  Pi = mod(Pi, rep.xyxy); // To create noise with explicit period
  Pi = mod289(Pi);        // To avoid truncation effects in permutation
  vec4 ix = Pi.xzxz;
  vec4 iy = Pi.yyww;
  vec4 fx = Pf.xzxz;
  vec4 fy = Pf.yyww;

  vec4 i = permute(permute(ix) + iy);

  vec4 gx = fract(i * (1.0 / 41.0)) * 2.0 - 1.0 ;
  vec4 gy = abs(gx) - 0.5 ;
  vec4 tx = floor(gx + 0.5);
  gx = gx - tx;

  vec2 g00 = vec2(gx.x,gy.x);
  vec2 g10 = vec2(gx.y,gy.y);
  vec2 g01 = vec2(gx.z,gy.z);
  vec2 g11 = vec2(gx.w,gy.w);

  vec4 norm = taylorInvSqrt(vec4(dot(g00, g00), dot(g01, g01), dot(g10, g10), dot(g11, g11)));
  g00 *= norm.x;
  g01 *= norm.y;
  g10 *= norm.z;
  g11 *= norm.w;

  float n00 = dot(g00, vec2(fx.x, fy.x));
  float n10 = dot(g10, vec2(fx.y, fy.y));
  float n01 = dot(g01, vec2(fx.z, fy.z));
  float n11 = dot(g11, vec2(fx.w, fy.w));

  vec2 fade_xy = fade(Pf.xy);
  vec2 n_x = mix(vec2(n00, n01), vec2(n10, n11), fade_xy.x);
  float n_xy = mix(n_x.x, n_x.y, fade_xy.y);
  return 2.3 * n_xy;
}

vec4 noise(vec2 tex_coords, float u_Scale, float u_S_factor, float u_T_factor) {

  float percent = ((1.0 + pnoise(u_Scale * tex_coords, vec2(u_S_factor, u_T_factor))) / 2.0);

  return vec4(percent, percent, percent, 1.0);
}

// function prototypes
vec3 CalcPointLight(PointLight light, vec3 normal, vec3 fragPos, vec3 viewDir);


void main()
{    
    // properties
    vec3 norm;
    if (textureSize(texture_normal1, 0).x > 1){
        norm = normalize(texture(texture_normal1, TexCoord).rgb * 2.0 - 1.0);
        norm = normalize(TBN * norm);
    }
    else {
        norm = normalize(Normal);
    }

    
    vec3 viewDir = normalize(viewPos - FragPos);
    
    // == =====================================================
    // Our lighting is set up in 3 phases: directional, point lights and an optional flashlight
    // For each phase, a calculate function is defined that calculates the corresponding color
    // per lamp. In the main() function we take all the calculated colors and sum them up for
    // this fragment's final color.
    // == =====================================================
    // phase 1: directional lighting
    vec3 result = vec3(0.0,0.0,0.0);
    // phase 2: point lights
    for(int i = 0; i < numLights; i++)
        result += CalcPointLight(pointLights[i], norm, FragPos, viewDir);    
    // phase 3: spot light
   // result += CalcSpotLight(spotLight, norm, FragPos, viewDir);
    result = result * objectColor;
    if (textureSize(texture_diffuse1, 0).x > 1){
            //FragColor = texture(texture_diffuse1, TexCoord) * vec4(result, 1.0);

            float min = 19;
            float max = 20;
            float scale = time*(max-min)+min;
            
            FragColor =  mix(texture(texture_diffuse1, TexCoord) , noise(TexCoord,scale,0,0), time*0.4) * vec4(result, 1.0);

    }
    else {
        FragColor = vec4(result, 1.0);
    }    
    
}


// calculates the color when using a point light.
vec3 CalcPointLight(PointLight light, vec3 normal, vec3 fragPos, vec3 viewDir)
{
    vec3 lightDir = normalize(light.position - fragPos);
    // diffuse shading
    float diff = max(dot(normal, lightDir), 0.0);
    // specular shading
    vec3 reflectDir = reflect(-lightDir, normal);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), 32.0);
    // attenuation
    float pdistance = length(light.position - fragPos);
    float attenuation = 1.0 / (light.constant + light.linear * pdistance + light.quadratic * (pdistance * pdistance));    
    // combine results
    vec3 ambient = light.ambient;
    vec3 diffuse = light.diffuse * diff * light.lightColor;
    vec3 specular = light.specular * spec * light.lightColor;
    ambient *= attenuation;
    diffuse *= attenuation;
    specular *= attenuation;
    return (ambient + diffuse + specular);
}