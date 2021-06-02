#version 410 core
out vec4 FragColor;
in vec2 TexCoord;
uniform vec3 objectColor;
uniform sampler2D texSampler;
void main()
{
    if (textureSize(texSampler, 0).x > 1){
        FragColor = texture(texSampler, TexCoord) * vec4(objectColor, 1.0);
    }
    else {
        FragColor = vec4(objectColor, 1.0);
    }    
}